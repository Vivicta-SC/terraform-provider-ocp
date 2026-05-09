.PHONY: all
all: tools headers fmt docs

.PHONY: tools
tools:
	go install github.com/hashicorp/copywrite
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

.PHONY: headers
headers:
	copywrite headers -d . --config .copywrite.hcl

.PHONY: fmt
fmt:
	terraform fmt -recursive examples/

.PHONY: docs
docs:
	tfplugindocs generate
