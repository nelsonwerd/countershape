.PHONY: test web release smoke

test:
	go test ./...

web:
	npm --prefix web run build

release: web
	./scripts/build-release.sh ./dist/release

smoke: release
	./scripts/package-smoke.sh ./dist/release/countershape-darwin-arm64/countershape
