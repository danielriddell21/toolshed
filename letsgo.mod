// letsgo.mod

// This repository's product is the collection, not any one tool. letsgo ships
// one archive per command, which for eleven commands means fifty-five
// downloads and eleven Homebrew formulas — and a formula named `fish` or
// `life` would shadow a Homebrew core package. This plugin answers the
// archive-layout hook with a single archive per target holding everything.
//
// Pinned by the digest of the released linux/amd64 binary, which is what the
// release runner installs: a program that decides what gets built is a build
// input exactly as the compiler is.
plugin archive-layout letsgo-multi v0.2.0 sha256:764a30483bd83f5f53074f9af9a3bb46b1893ec07cdeb61b5edef6b9698984be

// The six targets the GoReleaser config built for every command.
build (
	linux/amd64
	linux/arm64
	darwin/amd64
	darwin/arm64
	windows/amd64
	windows/arm64
)

// The version lives in a shared package rather than main, because eleven
// commands report it.
version github.com/danielriddell21/toolshed/internal/buildinfo.Version

brew danielriddell21/tap

// The shared GoReleaser workflow marked releases as pre-releases after
// publishing; letsgo does it while publishing, so promote.yaml still fires on
// manual promotion.
release prerelease=true
