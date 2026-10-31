'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const { parseMetadata, parsePrReferences, issueStatus, prStatus } = require('./project-sync-logic.cjs');

test('parses Roadmap and Priority metadata without relying on GitHub Milestone field', () => {
  assert.deepEqual(parseMetadata('**Roadmap:** M002 — Core architecture\n**Priority:** P0\n'), { milestone: 'M002', priority: 'P0' });
  assert.deepEqual(parseMetadata(' **Milestone:** M006 — Developer experience\r\n**Priority:** P1'), { milestone: 'M006', priority: 'P1' });
  assert.deepEqual(parseMetadata('**Roadmap:** future\n**Priority:** unknown'), { milestone: null, priority: null });
  assert.deepEqual(parseMetadata(null), { milestone: null, priority: null });
});

test('distinguishes PR references from closing keywords and deduplicates issues', () => {
  assert.deepEqual([...parsePrReferences('Refs #25\nCloses #4\nFixes #5\nResolves #6')], [[25, false], [4, true], [5, true], [6, true]]);
  assert.deepEqual([...parsePrReferences('Refs #4\nCloses #4\nRefs #4')], [[4, true]]);
  assert.deepEqual([...parsePrReferences('fixed #10, closed #11, fix #12, resolve #13')], [[10, true], [11, true], [12, true], [13, true]]);
  assert.deepEqual([...parsePrReferences('No issue references')], []);
});

test('an open issue referenced by a merged PR is not marked Done', () => {
  assert.equal(prStatus({ issueState: 'open', prState: 'closed', merged: true, closes: false, hasInProgress: true }), 'In Progress');
  assert.equal(prStatus({ issueState: 'open', prState: 'closed', merged: true, closes: false, hasInProgress: false }), null);
});

test('closing keyword and actually closed issues set Done', () => {
  assert.equal(prStatus({ issueState: 'open', prState: 'closed', merged: true, closes: true, hasInProgress: true }), 'Done');
  assert.equal(prStatus({ issueState: 'closed', prState: 'open', merged: false, closes: false, hasInProgress: true }), 'Done');
  assert.equal(prStatus({ issueState: 'open', prState: 'open', merged: false, closes: true, hasInProgress: true }), 'In Review');
});

test('draft, closed-unmerged, and reopened lifecycle transitions', () => {
  assert.equal(prStatus({ issueState: 'open', prState: 'open', merged: false, draft: true, hasInProgress: true }), 'In Progress');
  assert.equal(prStatus({ issueState: 'open', prState: 'closed', merged: false, draft: false, hasInProgress: true }), 'In Progress');
  assert.equal(prStatus({ issueState: 'open', prState: 'open', merged: false, draft: false, hasInProgress: true }), 'In Review');
  assert.equal(issueStatus('reopened', 'open'), 'Backlog');
  assert.equal(issueStatus('edited', 'open'), null);
  assert.equal(issueStatus('closed', 'closed'), 'Done');
});
