
fetch: deps
	go run crawler.go

deps:
	go mod download

lfs:
	git lfs track "*.zip"

zip:
	zip -r laws.json.zip laws.json

unzip:
	unzip laws.json.zip
