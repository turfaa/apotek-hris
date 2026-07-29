<!--
Title should follow Conventional Commits, e.g. feat(attendance): add quota audit log
-->

## Summary

<!-- What this changes and why, in a few sentences. -->

## AI attribution

<!--
Required. If any part of this change was written by an AI model, name the model here —
the same name used in the commit's Co-Authored-By trailer, e.g. "Claude Opus 5"
(see the co-author convention in CLAUDE.md).
Name every model if more than one was used. Write "None" if written entirely by hand.
-->

## Changes

<!-- One bullet per module or area touched, with a short note on what changed there. -->

-

## Verification

<!-- What you actually ran, and what it said. Note anything that could not be run here. -->

- [ ] `go build ./...`
- [ ] `go test ./...`
- [ ] `gofmt -l .` and `go vet ./...` clean
- [ ] `go run . migrate up` (and `down`) against a local database
- [ ] Exercised the affected endpoint or command

## Notes

<!-- Delete the lines that do not apply. -->

- **API**: endpoints added or changed — `docs/openapi.yaml` updated to match.
- **Migrations**: new files in `migrations/`, with a working `down`. Note anything not backwards compatible with the currently deployed server.
- **Money**: monetary values and multipliers use `shopspring/decimal`, never `float64`. Say how rounding is handled if it changed.
- **Salary**: which component type is affected (static / additional / dynamic) and whether existing snapshots stay valid.
- **Frontend**: paired `apotek-dashboard` change — link the PR.
