package main

import (
	"strconv"
	"testing"
)

func TestCleanAddress(t *testing.T) {
	tests := []struct {
		input    string
		omitName bool
		want     string
	}{
		{
			input:    "First Last <ok@example.com>",
			omitName: false,
			want:     "First Last <ok@example.com>",
		},
		{
			input:    "\"Google\" <google@reply.google.com>",
			omitName: false,
			want:     "Google <google@reply.google.com>",
		},
		{
			input:    "<register@example.com>",
			omitName: false,
			want:     "register@example.com",
		},
		{
			input:    "<REGISTER@example.com>",
			omitName: false,
			want:     "REGISTER@example.com",
		},
		{
			input:    "First Last <ok@example.com>",
			omitName: true,
			want:     "ok@example.com",
		},
		{
			input:    "\"Google\" <google@reply.google.com>",
			omitName: true,
			want:     "google@reply.google.com",
		},
		{
			input:    "<register@example.com>",
			omitName: true,
			want:     "register@example.com",
		},
		{
			input:    "<REGISTER@example.com>",
			omitName: true,
			want:     "REGISTER@example.com",
		},
		{
			input:    "",
			omitName: false,
			want:     "",
		},
		{
			input:    "foo",
			omitName: false,
			want:     "foo",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := cleanAddress(test.input, test.omitName)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}
