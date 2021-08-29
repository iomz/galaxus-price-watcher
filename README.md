angel-fetcher
---

This progam fetches the posts from girls in CityHeaven with a login credential.

# Prep

You need to download `geckodriver` ([link](https://github.com/mozilla/geckodriver)) and `selenium-server.jar` ([link](https://www.selenium.dev/downloads/)) in the `driver` directory.

The `db` directory stores the sqlite3 database files.

Edit `angel-fetcher.toml` to configure girls and the paths to the vendor files and the db directory.

# Build
```
go mod vendor
docker build --rm -t angel-fetcher:latest -f Dockerfile .
```

# Run
```
docker-compose up
```
