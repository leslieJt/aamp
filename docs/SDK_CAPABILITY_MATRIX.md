# SDK capability matrix

This document records the public capability surface of the SDKs in this
repository. It is a review checklist, not a requirement that every
language expose identical convenience APIs.

Legend: ✅ supported · ⚠️ intentionally SDK-specific · ❌ not provided

| Capability | Node.js | Python | Go | Notes |
| --- | --- | --- | --- | --- |
| Parse core task intents | ✅ | ✅ | ✅ | `task.dispatch`, `task.cancel`, `task.ack`, `task.help_needed`, `task.result`, and `task.stream.opened` |
| Send core task intents | ✅ | ✅ | ✅ | Available through the client or SMTP sender |
| Parse and send `pair.request` / `pair.respond` | ✅ | ✅ | ✅ | Includes pair code and dispatch-context-rules headers |
| Parse and send `card.query` / `card.response` | ✅ | ✅ | ✅ | Card summary is carried in `X-AAMP-Card-Summary` |
| Dispatch `sessionKey` | ✅ | ✅ | ✅ | Maps to `X-AAMP-Session-Key` |
| Dispatch context and parent task ID | ✅ | ✅ | ✅ | Header build and parse support |
| Structured task results | ✅ | ✅ | ✅ | Base64url JSON in `X-AAMP-StructuredResult` |
| Discovery and mailbox registration | ✅ | ✅ | ✅ | `/.well-known/aamp` plus registration and credential exchange |
| JMAP push with polling fallback | ✅ | ✅ | ✅ | Includes recent-mail reconciliation |
| Attachment send, receive, and blob download | ✅ | ✅ | ✅ | Public types differ by language |
| Stream create, append, get, and close | ✅ | ✅ | ✅ | Uses the discovered stream capability |
| Directory list, search, and profile update | ✅ | ✅ | ✅ | Uses discovered API actions |
| Pairing lifecycle helpers | ✅ | ❌ | ❌ | Node.js provides code creation, URL parsing, consumption, and sender-policy helpers |
| `registered-command/v1` dispatch helper | ✅ | ❌ | ❌ | Intentional Node.js convenience API; the wire message remains `task.dispatch` |
| Shared parser conformance fixtures | ✅ | ✅ | ✅ | All SDKs load `conformance/fixtures/parser.json` in their normal test suite |

## Conformance contract

Parser behavior shared across languages is defined by
[`conformance/fixtures/parser.json`](../conformance/fixtures/parser.json). The
canonical expected output uses lower-camel-case field names matching the
protocol types documented by the Node.js SDK. Language adapters may expose
additional fields, but must match every canonical field in a fixture.

The initial corpus covers:

- minimal `task.dispatch` defaults;
- dispatch session, context, expiry, priority, and parent-task headers;
- `pair.request` with dispatch-context rules;
- rejected `pair.respond`;
- `task.result` with a structured result;
- `task.help_needed` body parsing.

## Maintenance rule

Any pull request that changes a public SDK API or parser/header behavior must:

1. update this matrix when capability support changes;
2. update the shared fixtures when wire parsing changes;
3. run the Node.js, Python, and Go SDK test suites.

Node-only conveniences may remain intentional exceptions, but they must stay
visible in this matrix.
