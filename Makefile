# --- 변수 정의 ---
PROFILE := ndns
FUNCTION_NAME := ndns-tesseract
REGION := ap-northeast-2
ECR_PRIVATE_REPO_NAME := tesseract-go-lambda-repo
IMAGE_TAG := latest
ROLE_ARN := arn:aws:iam::323502797000:role/lambda_ndns_tesseract_role # 이 ARN은 공개해도 비교적 안전하지만, 민감하다고 판단하면 환경변수로 관리 가능

# Lambda 설정
TIMEOUT := 120
MEMORY := 1024
ARCHITECTURES := x86_64

# AWS CLI가 설치되어 있고 설정되어 있는지 확인
AWS_CLI := $(shell which aws) --profile $(PROFILE)
ifndef AWS_CLI
$(error "AWS CLI is not installed or not in your PATH. Please install it: https://aws.amazon.com/cli/")
endif

# Docker BuildX가 설치되어 있는지 확인
DOCKER_BUILDX := $(shell which docker buildx)
ifndef DOCKER_BUILDX
$(error "Docker Buildx is not installed. Please install it: https://docs.docker.com/build/install-buildx/")
endif


# --- PHONY 타겟 ---
.PHONY: all build-image push-image deploy-function create-function delete-function clean login-ecr-private login-ecr-public get-private-ecr-uri

# 기본 타겟
all: deploy-function

# ECR 프라이빗 레지스트리 URI를 동적으로 가져오는 타겟
get-private-ecr-uri:
	@$(eval ECR_FULL_URI := $(shell $(AWS_CLI) ecr describe-repositories --repository-names $(ECR_PRIVATE_REPO_NAME) --region $(REGION) --query 'repositories[0].repositoryUri' --output text))
	@echo "ECR_FULL_URI=$(ECR_FULL_URI)"

# ECR 프라이빗 레지스트리 로그인
login-ecr-private:
	@echo "--- Logging into private ECR registry ---"
	$(AWS_CLI) ecr get-login-password --region $(REGION) | docker login --username AWS --password-stdin $(shell echo $(ECR_FULL_URI) | cut -d/ -f1)

# ECR 퍼블릭 레지스트리 로그인 (필요한 경우)
login-ecr-public:
	@echo "--- Logging into public ECR registry ---"
	$(AWS_CLI) ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws

# Docker 이미지 빌드
build-image: get-private-ecr-uri 
	@echo "--- Building Docker image: $(ECR_FULL_URI):$(IMAGE_TAG) ---"
	$(DOCKER_BUILDX) build --platform linux/amd64 -t $(ECR_FULL_URI):$(IMAGE_TAG) -f Dockerfile.lambda .

# Docker 이미지 ECR에 푸시
push-image: build-image login-ecr-private
	@echo "--- Pushing Docker image to private ECR: $(ECR_FULL_URI):$(IMAGE_TAG) ---"
	docker push $(ECR_FULL_URI):$(IMAGE_TAG)

# Lambda 함수 업데이트
deploy-function: push-image
	@echo "--- Updating Lambda function code for $(FUNCTION_NAME) with image $(ECR_FULL_URI):$(IMAGE_TAG) ---"
	$(AWS_CLI) lambda update-function-code \
		--function-name $(FUNCTION_NAME) \
		--region $(REGION) \
		--image-uri $(ECR_FULL_URI):$(IMAGE_TAG)

	@echo "--- Updating Lambda function configuration for $(FUNCTION_NAME) ---"
	$(AWS_CLI) lambda update-function-configuration \
		--function-name $(FUNCTION_NAME) \
		--region $(REGION) \
		--timeout $(TIMEOUT) \
		--memory $(MEMORY) \
		--architectures $(ARCHITECTURES)
	@echo "Lambda function deployment (update) complete!"

# Lambda 함수 생성 (처음 한 번만 사용)
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
	@echo "Lambda function created!"

# Lambda 함수 삭제
delete-function:
	@echo "--- Deleting Lambda function: $(FUNCTION_NAME) ---"
	$(AWS_CLI) lambda delete-function --function-name $(FUNCTION_NAME) --region $(REGION) || true
	@echo "Lambda function $(FUNCTION_NAME) deleted."

# 모든 빌드 아티팩트 및 Docker 이미지 캐시 삭제
clean:
	@echo "--- Cleaning all build artifacts and Docker images ---"
	# 로컬 Docker 이미지 삭제. ECR_FULL_URI가 get-private-ecr-uri 타겟에 의해 설정된다고 가정
	$(eval ECR_FULL_URI := $(shell $(AWS_CLI) ecr describe-repositories --repository-names $(ECR_PRIVATE_REPO_NAME) --region $(REGION) --query 'repositories[0].repositoryUri' --output text))
	docker rmi $(ECR_FULL_URI):$(IMAGE_TAG) || true
	@echo "Clean complete."