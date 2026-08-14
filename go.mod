// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

module github.com/openchami/ochami

go 1.26.5

require (
	github.com/elliotchance/pie/v2 v2.9.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/knadh/koanf/parsers/yaml v1.1.1
	github.com/knadh/koanf/providers/file v1.2.1
	github.com/knadh/koanf/providers/rawbytes v1.0.1
	github.com/knadh/koanf/providers/structs v1.0.1
	github.com/knadh/koanf/v2 v2.3.6
	github.com/lestrrat-go/jwx/v3 v3.2.0
	github.com/nikolalohinski/gonja/v2 v2.9.0
	github.com/openchami/boot-service v0.3.2
	github.com/openchami/bss v1.32.3
	github.com/openchami/cloud-init v1.4.8
	github.com/openchami/fabrica v0.4.10
	github.com/openchami/metadata-service v0.2.2
	github.com/openchami/schemas v0.0.0-20250625220233-9aad17a286c4
	github.com/openchami/smd/v2 v2.20.5
	github.com/rs/zerolog v1.35.1
	github.com/spf13/cobra v1.10.2
	github.com/synackd/go-kargs v0.0.1-beta.1
	github.com/vbauerster/mpb/v8 v8.15.1
	golang.org/x/term v0.45.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Cray-HPE/hms-base v1.15.1 // indirect
	github.com/Cray-HPE/hms-base/v2 v2.3.0 // indirect
	github.com/Cray-HPE/hms-certs v1.7.1 // indirect
	github.com/Cray-HPE/hms-securestorage v1.18.0 // indirect
	github.com/Cray-HPE/hms-xname v1.4.0 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-7 // indirect
	github.com/hashicorp/vault/api v1.20.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/jsonschema v0.14.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/dsig v1.3.0 // indirect
	github.com/lestrrat-go/dsig-secp256k1 v1.0.0 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc/v3 v3.0.6 // indirect
	github.com/lestrrat-go/option/v2 v2.0.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.27 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/valyala/fastjson v1.6.10 // indirect
	github.com/vbauerster/cupwriter v0.0.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.2 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/exp v0.0.0-20250819193227-8b4c13bb791b // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/time v0.12.0 // indirect
)
