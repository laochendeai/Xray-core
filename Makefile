.PHONY: build web clean

# Build the frontend and then the Go binary
build: web
	rm -rf app/webpanel/dist
	mkdir -p app/webpanel/dist
	cp -a web/dist/. app/webpanel/dist/
	go build -o xray ./main

# Build frontend only
web:
	cd web && npm install && npm run build

# Development build (Go only, with placeholder frontend)
build-dev:
	go build -o xray ./main

# Clean build artifacts
clean:
	rm -f xray
	rm -rf web/dist web/node_modules
	rm -rf app/webpanel/dist
	mkdir -p app/webpanel/dist
	echo '<!DOCTYPE html><html><body><h1>Frontend not built</h1></body></html>' > app/webpanel/dist/index.html
