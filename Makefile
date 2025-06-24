# Makefile

# --- 변수 정의 ---
PROFILE := ndns
FUNCTION_NAME := ndns-tesseract
REGION := ap-northeast-2 
LAYER_NAME := tesseract-layer
LAYER_DESCRIPTION := "Tesseract OCR layer with Korean language support"

# Layer ARN은 동적으로 가져옵니다
LAYER_ARN := $(shell aws lambda list-layer-versions --profile $(PROFILE) --layer-name $(LAYER_NAME) --query 'LayerVersions[0].LayerVersionArn' --output text)

# YOUR_LAMBDA_EXECUTION_ROLE_ARN 값을 실제 ARN으로 대체하세요.
# make get-arns 타겟을 실행하여 확인하거나 수동으로 입력하세요.
# 예시: arn:aws:iam::123456789012:role/lambda_ndns_tesseract_role
ROLE_ARN := arn:aws:iam::323502797000:role/lambda_ndns_tesseract_role

# Lambda 함수 배포 관련 파일 및 디렉토리
ZIP_FILE := lambda_function.zip
BUILD_DIR := build
BOOTSTRAP_NAME := bootstrap

# Tesseract Layer 관련 파일 및 디렉토리
TESSERACT_LAYER_DIR := tesseract-layer
TESSERACT_LAYER_ZIP := tesseract-layer.zip
S3_BUCKET_FOR_LAYERS := ndns-tesseract-s3


# 환경 변수
ENV_VARS_JSON_STRING := {\"TESSDATA_PREFIX\":\"/opt/share\",\"LD_LIBRARY_PATH\":\"/opt/lib\",\"ERROR_WEBHOOK_URL\":\"https://discord.com/api/webhooks/1386515993656037436/K9D3F6OjJgY867UhWJ20AMhWtqOjn2mkyW3xbENrdyghVUQFC_YiNFuf4-8AiREo9FFe\"}

# Lambda 설정
TIMEOUT := 30
MEMORY := 512
ARCHITECTURES := x86_64
RUNTIME := provided.al2

# AWS CLI가 설치되어 있고 설정되어 있는지 확인
AWS_CLI := $(shell which aws) --profile ndns
ifndef AWS_CLI
$(error "AWS CLI is not installed or not in your PATH. Please install it: https://aws.amazon.com/cli/")
endif

# --- PHONY 타겟 (실제 파일이 아님을 명시) ---
.PHONY: all build deploy update create clean get-arns tesseract-layer publish-tesseract-layer delete-layer-versions reset-layer

# 기본 타겟: 빌드 후 배포 (업데이트)
all: deploy

# Go Lambda 바이너리 빌드 타겟
build:
	@echo "--- Building Go Lambda binary ---"
	mkdir -p $(BUILD_DIR)/
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BOOTSTRAP_NAME) main.go
	@echo "--- Creating deployment package ($(ZIP_FILE)) ---"
	cd $(BUILD_DIR) && zip -r ../$(ZIP_FILE) $(BOOTSTRAP_NAME)
	@echo "Build complete: $(ZIP_FILE) created."

# Tesseract Layer ZIP 파일 생성 타겟
tesseract-layer:
	@echo "--- Creating Tesseract Layer ZIP file ($(TESSERACT_LAYER_ZIP)) ---"
	# tesseract_layer 디렉토리가 존재하고 내부에 필요한 파일들이 미리 준비되어 있어야 합니다.
	# (예: bin/tesseract, bin/liblept.so, share/tessdata/kor.traineddata 등)
	cd $(TESSERACT_LAYER_DIR) && zip -r ../$(TESSERACT_LAYER_ZIP) .
	cd .. # 다시 프로젝트 루트로 돌아옴
	@echo "Tesseract Layer ZIP created: $(TESSERACT_LAYER_ZIP)"

build-tesseract-layer:
	@echo "--- Deleting all versions of $(LAYER_NAME) ---"
	$(AWS_CLI) lambda list-layer-versions --layer-name $(LAYER_NAME) --query 'LayerVersions[].Version' --output text | tr '\t' '\n' | xargs -I {} -r $(AWS_CLI) lambda delete-layer-version --layer-name $(LAYER_NAME) --version-number {}
	@echo "All layer versions deleted."
	@echo "--- Building Tesseract Layer Docker image $(ARCHITECTURES) ---"
	docker buildx build --platform linux/amd64 --load -t $(LAYER_NAME) -f Dockerfile.layer .
	@echo "--- Creating Lambda Layer .zip file from Docker image ---"
	docker run --rm -v $(PWD)/out:/out \
	           --user $(id -u):$(id -g) \
	           --entrypoint /opt/bin/zip $(LAYER_NAME) \
	           -r /out/$(TESSERACT_LAYER_ZIP) /opt/
	@echo "Lambda Layer .zip file created at out/$(TESSERACT_LAYER_ZIP)"

# publish-tesseract-layer: build-tesseract-layer
# 	@echo "--- Publishing Tesseract Layer to AWS Lambda ---"
# 	$(AWS_CLI) lambda publish-layer-version --layer-name $(LAYER_NAME) --description "Tesseract OCR with Korean language for Lambda" --zip-file fileb://out/$(TESSERACT_LAYER_ZIP) --compatible-runtimes python3.9 python3.10 python3.11 go1.x --region $(REGION) --compatible-architectures arm64
# 	@echo "Layer published. Remember to attach it to your Lambda Function."

publish-tesseract-layer: build-tesseract-layer
	@echo "--- Uploading Tesseract Layer .zip to S3 ---"
	$(AWS_CLI) s3 cp out/$(TESSERACT_LAYER_ZIP) s3://$(S3_BUCKET_FOR_LAYERS)/$(TESSERACT_LAYER_ZIP) --profile $(PROFILE) --region $(REGION)
	@echo "--- Publishing Tesseract Layer to AWS Lambda from S3 ---"
	$(AWS_CLI) lambda publish-layer-version \
		--layer-name $(LAYER_NAME) \
		--description $(LAYER_DESCRIPTION) \
		--content S3Bucket=$(S3_BUCKET_FOR_LAYERS),S3Key=$(TESSERACT_LAYER_ZIP) \
		--compatible-runtimes python3.9 python3.10 python3.11 go1.x \
		--region $(REGION) \
		--runtime go1.x \
		--compatible-architectures $(ARCHITECTURES)

delete-tesseract-layer:
	@echo "--- Deleting all versions of $(LAYER_NAME) ---"
	$(AWS_CLI) lambda list-layer-versions --layer-name $(LAYER_NAME) --query 'LayerVersions[].Version' --output text | tr '\t' '\n' | xargs -I {} -r $(AWS_CLI) lambda delete-layer-version --layer-name $(LAYER_NAME) --version-number {}
	@echo "All layer versions deleted."


# Go Lambda 함수 배포(업데이트) 타겟
deploy: build
	@echo "--- Updating Lambda function code for $(FUNCTION_NAME) ---"
	aws lambda update-function-code \
		--profile $(PROFILE) \
		--function-name $(FUNCTION_NAME) \
		--zip-file fileb://$(ZIP_FILE)

	@echo "--- Getting latest layer version ARN ---"
	$(eval LAYER_ARN := $(shell aws lambda list-layer-versions --profile $(PROFILE) --layer-name $(LAYER_NAME) --query 'LayerVersions[0].LayerVersionArn' --output text))
	@echo "Using layer ARN: $(LAYER_ARN)"

	@echo "--- Updating Lambda function configuration for $(FUNCTION_NAME) ---"
	aws lambda update-function-configuration \
		--profile $(PROFILE) \
		--function-name $(FUNCTION_NAME) \
		--layers $(LAYER_ARN) \
		--timeout $(TIMEOUT) \
		--memory $(MEMORY) \
		--runtime go1.x \
		--environment "{\"Variables\": $(ENV_VARS_JSON_STRING)}"
	@echo "Deployment (update) complete!"

# Go Lambda 함수 생성 타겟 (처음 한 번만 사용)
create: build
	@echo "--- Creating Lambda function $(FUNCTION_NAME) ---"
	aws lambda create-function \
		--profile $(PROFILE) \
		--function-name $(FUNCTION_NAME) \
		--runtime $(RUNTIME) \
		--handler $(BOOTSTRAP_NAME) \
		--role $(ROLE_ARN) \
		--zip-file fileb://$(ZIP_FILE) \
		--layers $(LAYER_ARN) \
		--timeout $(TIMEOUT) \
		--memory $(MEMORY) \
		--environment "{\"Variables\": $(ENV_VARS_JSON_STRING)}" \
		--architectures $(ARCHITECTURES)	
	@echo "Lambda function created!"

# 모든 빌드 아티팩트 삭제 타겟 (Go 바이너리 ZIP, Tesseract Layer ZIP)
clean:
	@echo "--- Cleaning all build artifacts ---"
	rm -rf $(BUILD_DIR) $(ZIP_FILE) $(TESSERACT_LAYER_ZIP)
	@echo "Clean complete."

# IAM Role 및 Layer ARN 확인 유틸리티 타겟
get-arns:
	@echo "--- Getting IAM Role ARN for $(ROLE_ARN) ---"
	aws iam get-role --profile $(PROFILE) --role-name lambda_ndns_tesseract_role --query 'Role.Arn' --output text

	@echo "--- Getting Tesseract Layer ARN for $(LAYER_ARN) ---"
	aws lambda list-layer-versions \
	  --profile $(PROFILE) \
	  --layer-name tesseract-go-layer \
	  --query 'LayerVersions[0].LayerVersionArn' \
	  --output text

