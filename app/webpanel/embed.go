package webpanel

import "embed"

// distFiles holds the compiled Vue frontend files.
// When building for development without the frontend, this will be empty.
// For production builds, run "cd web && npm run build" first, then
// copy the output to app/webpanel/dist/ before building Go.
//
//go:embed all:dist
var distFiles embed.FS
