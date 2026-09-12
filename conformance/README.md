# AAMP SDK conformance fixtures

The files in `fixtures/` are the language-neutral source of truth for behavior
that must stay aligned across the Node.js, Python, and Go protocol parsers.

Each case contains:

- `input`: email metadata passed to the SDK parser
- `expected`: the canonical lower-camel-case projection of the parsed message

The canonical field names follow the Node.js protocol types. An SDK may expose
additional language-specific fields, but every field present in `expected` must
be returned with the same value. Fields that are irrelevant to a case are
omitted instead of being represented as `null` or a language-specific zero
value.

When a public parser or protocol header changes:

1. update or add a fixture in `fixtures/parser.json`;
2. run all three SDK test suites;
3. update `docs/SDK_CAPABILITY_MATRIX.md` if the public capability surface
   changed.

Run the conformance tests through the normal SDK test commands:

```bash
cd packages/sdks/nodejs && npm test
cd packages/sdks/python && python -m unittest discover -s tests -v
cd packages/sdks/go && go test ./...
```
