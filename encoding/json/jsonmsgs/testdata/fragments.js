// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// A reference implementation, in JavaScript, of the fragmentation format
// documented in package jsonmsgs, written the way a browser extension would
// write it, that is using JSON.parse and JSON.stringify and nothing else, and
// used to test the Go implementation against an independent one.
//
// Usage: node fragments.js <budget> [repeat] < frames > frames
//
// Reads frames (4 byte little endian length followed by one JSON value) from
// stdin, reassembles the messages they carry, and writes each message back to
// stdout, fragmenting it into frames of at most <budget> bytes if need be. A
// message that has a user header, the "h" member, is written with it, on
// every fragment if "repeat" is given and on the first otherwise.

"use strict";

const PREFIX = '{"~":[';

// Reassembler returns {text, header} for a complete message when given the
// text of the frame that completes it, and undefined otherwise. header is
// undefined if the message has none.
class Reassembler {
  constructor() { this.reset(); }
  reset() { this.parts = []; this.total = 0; this.got = 0; this.header = undefined; }
  onFrame(text) {
    const m = JSON.parse(text);
    const h = (m !== null && typeof m === "object") ? m["~"] : undefined;
    if (!Array.isArray(h)) {
      if (this.parts.length) throw new Error("bare frame inside a message");
      return {text, header: undefined};
    }
    if (h[0] !== this.parts.length) throw new Error("bad sequence number");
    if (h[0] === 0) { this.total = h[1]; this.header = m.h; }
    else if (m.h !== undefined && JSON.stringify(m.h) !== JSON.stringify(this.header)) throw new Error("header changed");
    this.parts.push(m.p);
    this.got += Buffer.byteLength(m.p);
    if (this.got > this.total) throw new Error("too much data");
    if (this.got < this.total) return undefined;
    const out = {text: this.parts.join(""), header: this.header};
    this.reset();
    return out;
  }
}

// fragment returns the frame bodies that carry text, and header if it is not
// undefined, in frames of at most budget bytes.
function fragment(text, budget, header, repeat) {
  const total = Buffer.byteLength(text);
  const hasHeader = header !== undefined;
  const hlen = hasHeader ? Buffer.byteLength(JSON.stringify(header)) : 0;
  if (total <= budget && !hasHeader && !text.startsWith(PREFIX)) return [text];
  const out = [];
  const cps = Array.from(text);  // iterates by code point, never splitting a surrogate pair
  for (let i = 0, seq = 0; i < cps.length; seq++) {
    const withHeader = hasHeader && (seq === 0 || repeat);
    const hdr = seq === 0 ? [0, total] : [seq];
    if (withHeader) hdr.push(hlen);
    const env = (p) => withHeader ? {"~": hdr, h: header, p} : {"~": hdr, p};
    const empty = JSON.stringify(env(""));
    let room = budget - Buffer.byteLength(empty), chunk = "";
    for (; i < cps.length; i++) {
      const e = Buffer.byteLength(JSON.stringify(cps[i])) - 2;
      if (e > room) break;
      room -= e;
      chunk += cps[i];
    }
    if (chunk === "") throw new Error("budget too small");
    out.push(JSON.stringify(env(chunk)));
  }
  return out;
}

function frameOf(text) {
  const body = Buffer.from(text, "utf8");
  const hdr = Buffer.alloc(4);
  hdr.writeUInt32LE(body.length);
  return Buffer.concat([hdr, body]);
}

function main() {
  const budget = parseInt(process.argv[2], 10);
  const repeat = process.argv[3] === "repeat";
  const chunks = [];
  process.stdin.on("data", (c) => chunks.push(c));
  process.stdin.on("end", () => {
    const input = Buffer.concat(chunks);
    const re = new Reassembler();
    const out = [];
    for (let off = 0; off < input.length;) {
      const n = input.readUInt32LE(off);
      const text = input.toString("utf8", off + 4, off + 4 + n);
      off += 4 + n;
      const msg = re.onFrame(text);
      if (msg === undefined) continue;
      for (const f of fragment(msg.text, budget, msg.header, repeat)) out.push(frameOf(f));
    }
    process.stdout.write(Buffer.concat(out));
  });
}

if (require.main === module) main();
module.exports = {Reassembler, fragment};
