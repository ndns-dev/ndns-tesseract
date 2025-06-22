# Makefile

# --- 변수 정의 ---
PROFILE := ndns
FUNCTION_NAME := ndns-tesseract
REGION := ap-northeast-2 # 필요시 var.region처럼 변수로 분리 가능

# YOUR_TESSERACT_LAYER_ARN 값을 실제 ARN으로 대체하세요.
# make get-arns 타겟을 실행하여 확인하거나 수동으로 입력하세요.
# 예시: arn:aws:lambda:ap-northeast-2:123456789012:layer:tesseract-go-layer:1
LAYER_ARN := arn:aws:lambda:ap-northeast-2:323502797000:layer:tesseract-go-layer:1

# YOUR_LAMBDA_EXECUTION_ROLE_ARN 값을 실제 ARN으로 대체하세요.
# make get-arns 타겟을 실행하여 확인하거나 수동으로 입력하세요.
# 예시: arn:aws:iam::123456789012:role/lambda_ndns_tesseract_role
ROLE_ARN := arn:aws:iam::323502797000:role/lambda_ndns_tesseract_role

# Lambda 함수 배포 관련 파일 및 디렉토리
ZIP_FILE := lambda_function.zip
BUILD_DIR := build
BOOTSTRAP_NAME := bootstrap

# Tesseract Layer 관련 파일 및 디렉토리
TESSERACT_LAYER_DIR := tesseract_layer
TESSERACT_LAYER_ZIP := tesseract_layer.zip

# 환경 변수 (쉼표로 구분, 중괄호는 CLI에서 처리)
ENV_VARS := "TESSDATA_PREFIX=/opt/share,ERROR_WEBHOOK_URL=https://discord.com/api/webhooks/1386267994648477707/Vq9PdkFPm2qa1-hedexV3vNGxF5C7XIWOFupg_eisv6fAMYEyqaqvvEHAHdFInjE1mIO"

# Lambda 설정
TIMEOUT := 30
MEMORY := 512
ARCHITECTURES := x86_64
RUNTIME := provided.al2

# --- PHONY 타겟 (실제 파일이 아님을 명시) ---
.PHONY: all build deploy update create clean get-arns tesseract-layer publish-tesseract-layer

# 기본 타겟: 빌드 후 배포 (업데이트)
all: deploy

# Go Lambda 바이너리 빌드 타겟
build:
	@echo "--- Building Go Lambda binary ---"
	mkdir -p $(BUILD_DIR)/
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BOOTSTRAP_NAME) main.go
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

# Tesseract Layer를 AWS에 발행 (업데이트)하는 타겟
# Layer는 버전이 있기 때문에, 매번 publish하면 새 버전이 생성됩니다.
# Lambda 함수는 특정 Layer 버전을 바라보므로, Layer 업데이트 후에는 Lambda 함수 설정도 업데이트해야 할 수 있습니다.
publish-tesseract-layer: tesseract-layer
	@echo "--- Publishing Tesseract Layer to AWS ---"
	aws lambda publish-layer-version \
		--profile $(PROFILE) \
		--layer-name tesseract-go-layer \
		--description "Tesseract OCR binary and libraries for Go Lambda" \
		--zip-file fileb://$(TESSERACT_LAYER_ZIP) \
		--compatible-runtimes $(RUNTIME) \
		--compatible-architectures $(ARCHITECTURES) \
		--query 'LayerVersionArn' \
		--output text
	@echo "Tesseract Layer published. Remember to update LAYER_ARN in this Makefile if the version changed."

# Go Lambda 함수 배포(업데이트) 타겟
deploy: build
	@echo "--- Updating Lambda function code for $(FUNCTION_NAME) ---"
	aws lambda update-function-code \
		--profile $(PROFILE) \
		--function-name $(FUNCTION_NAME) \
		--zip-file fileb://$(ZIP_FILE)

	@echo "--- Updating Lambda function configuration for $(FUNCTION_NAME) ---"
	aws lambda update-function-configuration \
		--profile $(PROFILE) \
		--function-name $(FUNCTION_NAME) \
		--layers $(LAYER_ARN) \
		--timeout $(TIMEOUT) \
		--memory $(MEMORY) \
		--environment Variables=$(ENV_VARS) \
		--architectures $(ARCHITECTURES)
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
		--environment Variables=$(ENV_VARS) \
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