# Release process

Before beginning, you will need your Github username and password or personal access token.

```bash
export GITHUB_USERNAME=$YOUR_GH_USERNAME
make clean
make release/tag/local
make release/tag/push
make release/github/draft
```

At this point, you should inspect the release in the Github web UI. If it looks reasonable, proceed:

```bash
make release/github/publish
```
