# Contributing to CropDoc

Thank you for your interest. CropDoc is an open, Sri Lanka-first agricultural knowledge project. We welcome contributions from farmers, agronomists, extension officers, researchers, translators, and developers.

## Ways to contribute

- **Corpus corrections** — spot an error in a crop, disease, variety, or remedy record? Open an issue or PR against the relevant file in `corpus/`. Cite the source.
- **New sources** — know a public, permissively licensed source we aren't crawling? Open an issue proposing it; include the URL, licence, and why it's reliable.
- **Translations** — help bring Sinhala and Tamil to parity with English.
- **Schemas** — propose additions or refinements to the domain model via an issue first, then a PR.
- **Crawlers / pipelines** — add a new source crawler or improve an existing extractor.
- **Apps** — web, mobile, admin CMS.
- **Docs** — improve any document in `docs/`.

## Ground rules

- **Cite your sources.** Any PR touching corpus content must reference the source URL(s) in the description.
- **Respect source terms.** We redistribute structured extractions with attribution. We link out to third-party media; we do not redistribute copyrighted videos or images.
- **No scraping behind logins, paywalls, or rate-limit circumvention.** Crawlers must honour `robots.txt`, identify themselves, and rate-limit politely.
- **Agronomic safety.** Changes affecting chemical dosages, PHI, disease diagnoses, or direct recommendations to farmers require review by a qualified agronomist before merge. Flag your PR with the `needs-agronomist-review` label.
- **No personal data.** Do not include real users' plots, identities, or private content in examples or fixtures.

## Workflow

1. **Open an issue** before substantial work. For small corrections, a direct PR is fine.
2. **Branch naming**:
   - `feat/<area>-<short-desc>`
   - `fix/<area>-<short-desc>`
   - `data/<source>-<short-desc>` for corpus/crawler changes
   - `docs/<short-desc>`
3. **Commit messages** — imperative mood, concise (e.g., `fix: correct PHI for Mancozeb on tomato`). Reference issues in the body.
4. **Pull requests** — describe what and why. Attach source links for data changes. Keep diffs reviewable; split large changes.
5. **CI must be green** — lint, typecheck, schema validation, and tests.

## Code style

- Follow the existing style in the package you're editing; repo-wide formatters (Prettier, ruff, etc.) are configured.
- Type-check everything. TypeScript `strict`, Python `mypy --strict` where feasible.
- Write tests for extractors and schema-changing code.

## Adding a new source

Any new crawler needs, in its PR:

1. **Source register entry** — name, URL, licence, `robots.txt` compliance check date, rate limit, expected freshness.
2. **Crawler module** under `pipelines/<source>/` with `fetch`, `discover`, `fingerprint`, tests, and fixtures.
3. **Extractor** declaring input content-type and output schema.
4. **Fixtures** — small, representative raw samples plus expected extracted output.
5. **Documentation** — a short README in the crawler folder covering failure modes and how to re-run.

## Reporting bugs & data errors

Open an issue with:

- What you expected, and what happened.
- Affected record(s) by slug or ID.
- Source URL supporting the correct value.
- Device / OS / app version if a client bug.

## Code of Conduct

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Be kind. Assume good faith. Disagree on the merits.

## Licences

By contributing, you agree that:

- Code contributions are licensed under the repository's [MIT License](LICENSE).
- Content contributions (docs, corpus records) are licensed under **CC-BY-SA 4.0**.
- Geodata contributions are licensed under **ODbL**.
- Images you upload are licensed under **CC-BY 4.0** and you have the right to license them.

## Questions

Open a GitHub Discussion or issue. We'd rather answer a question than merge a misunderstanding.
