#!/usr/bin/env bash


set -e

echo n | npm create vite@latest webapp-sample-vite  -y -- --yes --template react-ts --rolldown
(cd webapp-sample-vite; npm install)
