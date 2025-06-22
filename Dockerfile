# --- Stage 1: Builder ---
# Go 애플리케이션을 빌드하기 위한 스테이지
FROM public.ecr.aws/lambda/go:1.x as builder

WORKDIR /app

# Go 모듈 파일 복사 및 의존성 다운로드
COPY go.mod go.sum ./
RUN go mod download

# Go 소스 코드 복사
COPY main.go .

# Go 애플리케이션 빌드 (Linux 환경용)
# CGO_ENABLED=0는 C 바인딩 없이 정적 컴파일을 하여 최종 바이너리를 더 독립적으로 만듭니다.
# 결과 바이너리 이름은 'main'으로 설정합니다.
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# --- Stage 2: Runtime Image ---
# 실제 Lambda 함수 런타임에 사용될 최종 이미지
# Tesseract OCR 바이너리와 언어 데이터를 포함해야 합니다.
# Amazon Linux 2 (AL2) 기반 이미지가 Lambda와 호환성이 좋습니다.
FROM public.ecr.aws/lambda/go:1.x

# Tesseract OCR 및 필요한 라이브러리 설치
# 'tesseract-langpack-eng' 및 'tesseract-langpack-kor'는 영어/한글 언어 팩입니다.
# 'libjpeg-turbo-devel', 'libpng-devel', 'libtiff-devel', 'zlib-devel' 등은
# Tesseract가 이미지 처리 시 의존하는 라이브러리들입니다.
# 설치 후 캐시를 정리하여 이미지 크기를 줄입니다.
RUN yum install -y \
    tesseract \
    tesseract-langpack-eng \
    tesseract-langpack-kor \
    libjpeg-turbo-devel \
    libpng-devel \
    libtiff-devel \
    zlib-devel \
    && yum clean all \
    && rm -rf /var/cache/yum

# 빌더 스테이지에서 컴파일된 Go 애플리케이션 바이너리 복사
COPY --from=builder /app/main ${LAMBDA_TASK_ROOT}/main

# TESSDATA_PREFIX 환경 변수 설정
# Tesseract가 언어 데이터 파일을 찾을 경로를 지정합니다.
# yum 설치 시 언어 팩은 일반적으로 /usr/share/tessdata 에 위치합니다.
ENV TESSDATA_PREFIX=/usr/share/tessdata

# Lambda 런타임에 의해 실행될 CMD (Go 바이너리 실행)
CMD [ "main" ]