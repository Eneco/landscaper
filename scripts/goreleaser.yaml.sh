# This shellscript generates goreleaser.yaml with ldflags set to the first argument.
cat <<EOF
build:
  main: ./cmd/
  ldflags: "$1"
  env:
   - CGO_ENABLED=0
  goos:
    - linux
    - darwin
  goarch:
    - amd64
    - 386
archive:
  name_template: "{{.Binary}}-{{.Version}}-{{.Os}}-{{.Arch}}"
  replacements:
    amd64: amd64
    darwin: darwin
    linux: linux
EOF
