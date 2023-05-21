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


.PHONY: update-kubernetes-proto
update-kubernetes-proto: proto-tools
	echo "You need to install protoc before running this."
	echo "Regenerate the protobuf definition from the submodule ./kubernetes/kubernetes."
	openapi2proto -spec ./kubernetes/kubernetes/api/openapi-spec/swagger.json -out ./kubernetes/proto/kubernetes.proto
	echo "Regenerate the Go protobuf code."
	protoc ./kubernetes/proto/kubernetes.proto --go_out=./kubernetes/proto --go_opt=Mkubernetes/proto/kubernetes.proto=./api
