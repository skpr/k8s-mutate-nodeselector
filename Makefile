#!/usr/bin/make -f

lint:
	revive -set_exit_status -exclude=./vendor/... ./...

.PHONY: *
