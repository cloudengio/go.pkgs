if (typeof chromedp_utils === "undefined") {
    chromedp_utils = {};

    // This is script is complicated by the desire to catch errors (exceptions)
    // that are encountered by the browser when it is parsing the downloaded
    // javascript. These errors can only be caught by listening for errors
    // on the top-level window and will occur after the download has
    // completed. Therefore the onload handler gives the browser time
    // (via the setTimeout 0) to process any errors and only then will it
    // return with an indication of success.
    chromedp_utils.loadScript = async function (script_url) {
        console.log("Starting script load:", script_url);
        const result = await new Promise((resolve) => {
            let resolved = false; let errorCaught = null;

            function finish(result) {
                if (!resolved) {
                    resolved = true;
                    window.removeEventListener("error", errorListener);
                    resolve(result);
                }
            }

            function errorListener(event) {
                console.log("errorListener: filename: ", event.filename);
                if (event.filename && event.filename.includes(script_url)) {
                    errorCaught = event.error || new Error(event.message);
                    finish({ success: false, error: String(errorCaught) });
                }
            }

            window.addEventListener("error", errorListener);

            const script = document.createElement('script');
            script.src = script_url;
            script.onload = () => {
                console.log("onload for: ", script_url);
                setTimeout(() => { // First macrotask (flush microtasks)
                    setTimeout(() => { // Second macrotask
                        if (!errorCaught) {
                            finish({ success: true, error: "" });
                        }
                    }, 0);
                }, 0);
            };

            script.onerror = (err) => {
                console.log("onerror for: ", script_url, err);
                finish({ success: false, error: "Failed to load script: " + script_url });
            };

            document.head.appendChild(script);

        });
        console.log("Script result: ", script_url, "result", result);
        return result;
    }

    chromedp_utils.safeClone = function safeClone(obj) {
        if (obj === null || typeof obj !== 'object') return obj;
        if (Array.isArray(obj)) return obj.map(safeClone);
        if (Object.getPrototypeOf(obj) !== Object.prototype) {
            const ctor = obj.constructor?.name || 'unknown';
            if (obj instanceof Element) {
                return {
                    _type: 'Element',
                    tagName: obj.tagName,
                    id: obj.id || null,
                    className: obj.className || null,
                    text: obj.textContent?.trim().slice(0, 100) || null
                };
            }
            if (obj instanceof Document) {
                return { _type: 'Document', title: obj.title || null, URL: obj.URL || null };
            }
            if (obj instanceof Window) {
                return { _type: 'Window', location: obj.location?.href || null };
            }
            if (obj instanceof Blob) {
                return { _type: 'Blob', size: obj.size, type: obj.type };
            }
            if (obj instanceof ArrayBuffer) {
                return { _type: 'ArrayBuffer', byteLength: obj.byteLength };
            }
            return { _type: 'PlatformObject', className: ctor };
        }
        const result = {};
        for (const key in obj) {
            try { result[key] = safeClone(obj[key]); }
            catch (e) { result[key] = "[unserializable: " + e + "]"; }
        }
        return result;
    }
}
