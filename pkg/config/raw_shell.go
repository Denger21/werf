package config

import (
	"fmt"
)

type RawShell struct {
	BeforeInstall interface{} `yaml:"beforeInstall,omitempty"`
	Install       interface{} `yaml:"install,omitempty"`
	BeforeSetup   interface{} `yaml:"beforeSetup,omitempty"`
	Setup         interface{} `yaml:"setup,omitempty"`
	BuildArtifact interface{} `yaml:"buildArtifact,omitempty"`

	UnsupportedAttributes map[string]interface{} `yaml:",inline"`
}

func (c *RawShell) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain RawShell
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if err := CheckOverflow(c.UnsupportedAttributes, c); err != nil {
		return err
	}

	return nil
}

func (c *RawShell) ToDirective() (shell *ShellDimg, err error) {
	shell = &ShellDimg{}
	shell.ShellBase = &ShellBase{}

	if beforeInstall, err := InterfaceToStringArray(c.BeforeInstall); err != nil {
		return nil, err
	} else {
		shell.ShellBase.BeforeInstall = beforeInstall
	}

	if install, err := InterfaceToStringArray(c.Install); err != nil {
		return nil, err
	} else {
		shell.ShellBase.Install = install
	}

	if beforeSetup, err := InterfaceToStringArray(c.BeforeSetup); err != nil {
		return nil, err
	} else {
		shell.ShellBase.BeforeSetup = beforeSetup
	}

	if setup, err := InterfaceToStringArray(c.Setup); err != nil {
		return nil, err
	} else {
		shell.ShellBase.Setup = setup
	}

	if c.BuildArtifact != nil {
		return nil, fmt.Errorf("директива buildArtifact не может быть объявлена для dimg-а!") // FIXME
	}

	return shell, nil
}

func (c *RawShell) ToArtifact() (shellArtifact *ShellArtifact, err error) {
	shellArtifact = &ShellArtifact{}

	if shellDimg, err := c.ToDirective(); err != nil {
		return nil, err
	} else {
		shellArtifact.ShellDimg = shellDimg
	}

	return shellArtifact, nil
}