.PHONY: install-tools
install-tools:
	@cd tools && cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI {} go install {}
