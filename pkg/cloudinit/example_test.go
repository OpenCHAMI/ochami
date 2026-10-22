// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/openchami/ochami/pkg/cloudinit"
	"github.com/openchami/ochami/pkg/format"
)

func ExampleRender() {
	out, err := cloudinit.Render(context.Background(), cloudinit.RenderRequest{
		Template:  []byte("#cloud-config\nhostname: {{ ds.meta_data.hostname }}\nrole: {{ role }}\n"),
		Metadata:  cloudinit.Document{Data: []byte("hostname: node01\n"), Format: format.DataFormatYaml},
		ExtraVars: cloudinit.Document{Data: []byte(`{"role":"compute"}`)},
	})
	if err != nil {
		panic(err)
	}
	fmt.Print(out)
	// Output:
	// #cloud-config
	// hostname: node01
	// role: compute
}

func ExampleRenderError() {
	_, err := cloudinit.Render(context.Background(), cloudinit.RenderRequest{Template: []byte("{% if")})
	var renderErr *cloudinit.RenderError
	if errors.As(err, &renderErr) {
		fmt.Println(renderErr.Stage)
	}
	// Output: compile
}
