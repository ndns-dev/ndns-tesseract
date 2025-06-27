# --- 변수 정의 ---
PROFILE := ndns
FUNCTION_NAME := TesseractGoOCRFunction
REGION := ap-northeast-2
ECR_PRIVATE_REPO_NAME := tesseract-go-lambda-repo
IMAGE_TAG := latest
ROLE_ARN := arn:aws:iam::323502797000:role/lambda_ndns_tesseract_role
AWS_ACCOUNT_ID := 323502797000
# Lambda 설정
TIMEOUT := 30
MEMORY := 1024
ARCHITECTURES := x86_64

# AWS CLI 경로 (환경에 따라 변경될 수 있음)
AWS_CLI := /usr/local/bin/aws --profile $(PROFILE)

# ECR URI
ECR_FULL_URI := $(AWS_ACCOUNT_ID).dkr.ecr.$(REGION).amazonaws.com/$(ECR_PRIVATE_REPO_NAME)

# --- Lambda 함수 생성 (최초 배포 시에만 사용) ---
create-function: push-image
	@echo "--- Creating Lambda function $(FUNCTION_NAME) with image $(ECR_FULL_URI):$(IMAGE_TAG) ---"
	$(AWS_CLI) lambda create-function \
		--function-name $(FUNCTION_NAME) \
		--package-type Image \
		--code ImageUri=$(ECR_FULL_URI):$(IMAGE_TAG) \
		--role $(ROLE_ARN) \
		--timeout $(TIMEOUT) \
		--memory $(MEMORY) \
		--architectures $(ARCHITECTURES) \
		--region $(REGION)

	@echo "Lambda function creation complete!"


# --- Lambda 함수 업데이트 ---
deploy-function: push-image
	@echo "--- Updating Lambda function code for $(FUNCTION_NAME) with image $(ECR_FULL_URI):$(IMAGE_TAG) ---"
	$(AWS_CLI) lambda update-function-code \
		--function-name $(FUNCTION_NAME) \
		--region $(REGION) \
		--image-uri $(ECR_FULL_URI):$(IMAGE_TAG)

	sleep 10

	@echo "--- Updating Lambda function configuration for $(FUNCTION_NAME) ---"
	$(AWS_CLI) lambda update-function-configuration \
		--function-name $(FUNCTION_NAME) \
		--region $(REGION) \
		--timeout $(TIMEOUT) \
		--memory $(MEMORY)

	@echo "Lambda function deployment (update) complete!"

# --- Docker 이미지 빌드 및 푸시 ---
push-image: build-image
	@echo "--- Logging into private ECR registry ---"
	$(AWS_CLI) ecr get-login-password --region $(REGION) | docker login --username AWS --password-stdin $(ECR_FULL_URI)

	@echo "--- Pushing Docker image to private ECR: $(ECR_FULL_URI):$(IMAGE_TAG) ---"
	docker push $(ECR_FULL_URI):$(IMAGE_TAG)

build-image:
	@echo "--- Building Docker image: $(ECR_FULL_URI):$(IMAGE_TAG) ---"
	/usr/bin/docker build --platform linux/amd64 -t $(ECR_FULL_URI):$(IMAGE_TAG) -f Dockerfile.lambda .

# --- Lambda 함수 삭제 ---
delete-function:
	@echo "--- Deleting Lambda function $(FUNCTION_NAME) ---"
	$(AWS_CLI) lambda delete-function \
		--function-name $(FUNCTION_NAME) \
		--region $(REGION)
	@echo "Lambda function deletion complete!"

# --- Docker 이미지 정리 ---
clean:
	@echo "--- Cleaning up Docker images and build artifacts ---"
	-docker rmi $(ECR_FULL_URI):$(IMAGE_TAG) || true
	-rm -f bootstrap
