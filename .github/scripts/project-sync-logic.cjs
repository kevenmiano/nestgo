'use strict';

// Pure mapping logic; no GitHub credentials or network calls here.
function metadataValue(body, key) {
  const prefix = ('**' + key + ':**').toLowerCase();
  for (const line of String(body || '').split(/\r?\n/)) {
    const text = line.trim();
    if (text.toLowerCase().startsWith(prefix)) {
      return text.slice(prefix.length).trim();
    }
  }
  return '';
}

function parseMetadata(body) {
  // The issue may call the milestone grouping "Roadmap" or "Milestone".
  // The GitHub Project destination field is deliberately named "Roadmap"
  // to avoid colliding with GitHub's native Milestone field.
  const roadmap = metadataValue(body, 'Roadmap') || metadataValue(body, 'Milestone');
  const priorityText = metadataValue(body, 'Priority');
  const milestone = roadmap.match(/\bM00[1-6]\b/i)?.[0].toUpperCase() || null;
  const priority = priorityText.match(/\bP[012]\b/i)?.[0].toUpperCase() || null;
  return { milestone, priority };
}

function parsePrReferences(body) {
  const references = new Map();
  const pattern = /\b(closes?|closed|fix(?:es|ed)?|resolv(?:e|es|ed)|refs?)\s+#(\d+)\b/gi;
  for (const [, keyword, rawNumber] of String(body || '').matchAll(pattern)) {
    const issueNumber = Number(rawNumber);
    const isClosingReference = !/^refs?$/i.test(keyword);
    // An explicit closing keyword takes precedence over a plain reference.
    references.set(issueNumber, (references.get(issueNumber) || false) || isClosingReference);
  }
  return references;
}

function issueStatus(action, state) {
  if (state === 'closed') return 'Done';
  if (action === 'opened' || action === 'reopened') return 'Backlog';
  return null;
}

function prStatus({ issueState, prState, merged, draft, closes, hasInProgress }) {
  if (issueState === 'closed') return 'Done';
  // GitHub issue state is authoritative: PR keywords alone never imply completion.
  // GitHub may not have closed the issue (e.g. cross-branch PR or delayed events).
  if (merged && closes) return hasInProgress ? 'In Progress' : null;
  // A partial PR (Refs #N) must never mark an open issue Done.
  // Use In Progress if the Project offers it, otherwise preserve Status.
  if (merged || prState === 'closed' || draft) {
    return hasInProgress ? 'In Progress' : null;
  }
  return 'In Review';
}

// Existing GitHub Projects often ship with Todo / In Progress / Done instead
// of a custom In Review option. Select the nearest safe existing status;
// never replace an in-progress state with Done or a backlog state.
function resolveStatusOption(desired, options) {
  const aliases = {
    Backlog: ['Backlog', 'Todo', 'To Do', 'To-do', 'Not Started'],
    'In Progress': ['In Progress', 'Doing', 'Working', 'Active'],
    'In Review': ['In Review', 'Review', 'In Progress', 'Doing', 'Working', 'Active'],
    Done: ['Done', 'Complete', 'Completed', 'Finished']
  };
  const values = aliases[desired] || [desired];
  for (const candidate of values) {
    const selected = (options || []).find(
      option => option.name.trim().toLowerCase() === candidate.toLowerCase()
    );
    if (selected) return selected;
  }
  return null;
}

module.exports = { parseMetadata, parsePrReferences, issueStatus, prStatus, resolveStatusOption };
