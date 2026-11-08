#!/usr/bin/env node
// Computes the next semver from the last git tag and commit-message markers.
// #major -> X+1.0.0, #minor -> X.Y+1.0, otherwise X.Y.Z+1. Prints without the leading "v".
import { execSync } from 'node:child_process';

const tags = execSync('git tag --list "v*" --sort=-v:refname')
  .toString()
  .trim()
  .split('\n')
  .filter(Boolean);
const last = tags[0] ?? 'v0.0.0';
const rev = tags.length ? `${last}..HEAD` : 'HEAD';
const msgs = execSync(`git log ${rev} --pretty=%B`).toString();

let [major, minor, patch] = last.slice(1).split('.').map(Number);
if (msgs.includes('#major')) [major, minor, patch] = [major + 1, 0, 0];
else if (msgs.includes('#minor')) [minor, patch] = [minor + 1, 0];
else patch += 1;

console.log(`${major}.${minor}.${patch}`);
