cat <<EOF
build:
  main: ./cmd/
  ldflags: "$1"
  goos:
    - linux
  goarch:
    - amd64
archive:
  name_template: "{{.Binary}}-{{.Version}}-{{.Os}}-{{.Arch}}"
  replacements:
    amd64: amd64
    darwin: darwin
    linux: linux
EOF