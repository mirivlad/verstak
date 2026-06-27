package externalopen

import (
	"reflect"
	"testing"
)

func TestOpenPathCommands(t *testing.T) {
	cases := []struct {
		goos string
		path string
		want commandCall
	}{
		{goos: "linux", path: "/vault/file.txt", want: commandCall{name: "xdg-open", args: []string{"/vault/file.txt"}}},
		{goos: "darwin", path: "/vault/file.txt", want: commandCall{name: "open", args: []string{"/vault/file.txt"}}},
		{goos: "windows", path: `C:\Vault\file.txt`, want: commandCall{name: "rundll32", args: []string{"url.dll,FileProtocolHandler", `C:\Vault\file.txt`}}},
	}

	for _, tc := range cases {
		t.Run(tc.goos, func(t *testing.T) {
			var got commandCall
			svc := NewServiceFor(tc.goos, func(name string, args ...string) error {
				got = commandCall{name: name, args: args}
				return nil
			})
			if err := svc.OpenPath(tc.path); err != nil {
				t.Fatalf("OpenPath: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("command = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestShowInFolderCommands(t *testing.T) {
	cases := []struct {
		name  string
		goos  string
		path  string
		isDir bool
		want  commandCall
	}{
		{name: "linux file opens parent", goos: "linux", path: "/vault/Docs/file.txt", want: commandCall{name: "xdg-open", args: []string{"/vault/Docs"}}},
		{name: "linux dir opens dir", goos: "linux", path: "/vault/Docs", isDir: true, want: commandCall{name: "xdg-open", args: []string{"/vault/Docs"}}},
		{name: "darwin reveal", goos: "darwin", path: "/vault/file.txt", want: commandCall{name: "open", args: []string{"-R", "/vault/file.txt"}}},
		{name: "windows select file", goos: "windows", path: `C:\Vault\file.txt`, want: commandCall{name: "explorer", args: []string{`/select,C:\Vault\file.txt`}}},
		{name: "windows open dir", goos: "windows", path: `C:\Vault\Docs`, isDir: true, want: commandCall{name: "explorer", args: []string{`C:\Vault\Docs`}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got commandCall
			svc := NewServiceFor(tc.goos, func(name string, args ...string) error {
				got = commandCall{name: name, args: args}
				return nil
			})
			if err := svc.ShowInFolder(tc.path, tc.isDir); err != nil {
				t.Fatalf("ShowInFolder: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("command = %#v, want %#v", got, tc.want)
			}
		})
	}
}

type commandCall struct {
	name string
	args []string
}
