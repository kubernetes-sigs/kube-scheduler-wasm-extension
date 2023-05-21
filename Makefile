.PHONY: submodule-update
submodule-update:
	git submodule update -i
	cp ./kubernetes/kubernetes.checkout ./.git/modules/kubernetes/kubernetes/info/sparse-checkout
	cd ./kubernetes/kubernetes; \
	git config core.sparsecheckout true; \
	git read-tree -mu HEAD


.PHONY: update-kubernetes-proto
	go install github.com/sanposhiho/openapi2proto/cmd/openapi2proto@kubernetes-compat
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	openapi2proto -spec ./kubernetes/kubernetes/api/openapi-spec/swagger.json -out ./kubernetes/proto/kubernetes.proto