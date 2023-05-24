.PHONY: proto-tools
proto-tools:
	cd ./kubernetes/proto/tools; \
	cat tools.go | grep "_" | awk -F'"' '{print $$2}' | xargs -tI % go install %

.PHONY: submodule-update
submodule-update:
	git submodule update -i
	cp ./kubernetes/kubernetes.checkout ./.git/modules/kubernetes/kubernetes/info/sparse-checkout
	cd ./kubernetes/kubernetes; \
	git config core.sparsecheckout true; \
	git read-tree -mu HEAD

# This uses go-plugin to generate UnmarshalVT because generating via below
# hangs compiling with TinyGo.
# --go-vtproto_out=./kubernetes/proto --go-vtproto_opt=Mkubernetes/proto/kubernetes.proto=./api,features=marshal+unmarshal+size
.PHONY: update-kubernetes-proto
update-kubernetes-proto: proto-tools
	echo "You need to install protoc before running this."
	echo "Regenerate the protobuf definition from the submodule ./kubernetes/kubernetes."
	openapi2proto -spec ./kubernetes/kubernetes/api/openapi-spec/swagger.json -out ./kubernetes/proto/kubernetes.proto
	echo "Regenerate the Go protobuf code."
	protoc ./kubernetes/proto/kubernetes.proto \
		--go-plugin_out=./kubernetes/proto --go-plugin_opt=Mkubernetes/proto/kubernetes.proto=./api
