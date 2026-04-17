## Summary

<!-- One-paragraph summary of the change, and the user-visible / operator-visible impact. -->

## Type of change

- [ ] Feature
- [ ] Bug fix
- [ ] Refactor / chore
- [ ] Corpus content (seed records, translations)
- [ ] Crawler / pipeline
- [ ] Docs

## For corpus content PRs

- [ ] Every published field carries `provenance` with a real `source_url` and `quote`
- [ ] Source URL(s) cited below and the site's robots / terms allow our use
- [ ] Sinhala + Tamil translations are present or deliberately deferred (note below)

Sources:

<!-- paste source URLs -->

## For crawler PRs

- [ ] `robots.txt` checked; cadence and User-Agent declared in config
- [ ] Rate limit ≤ 1 req/sec per host
- [ ] ETag / Last-Modified caching in place

## Test plan

- [ ] <!-- manual or automated checks performed -->
