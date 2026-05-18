.PHONY: all
all: tools headers fmt docs

.PHONY: tools
tools:
	go install github.com/hashicorp/copywrite@latest
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

.PHONY: headers
headers:
	copywrite headers -d . --config .copywrite.hcl

.PHONY: fmt
fmt:
	terraform fmt -recursive examples/

.PHONY: docs
docs:
	tfplugindocs generate
