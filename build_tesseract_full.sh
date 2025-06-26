#!/usr/bin/env bash
# build_tesseract_full.sh

# 오류 발생 시 즉시 스크립트 중단
set -e

# 초기 위치를 /tmp로 설정 (Dockerfile의 WORKDIR과 일치)
cd /tmp

# Leptonica 빌드
echo "--- Leptonica 빌드 시작 ---"
git clone https://github.com/DanBloomberg/leptonica.git
cd leptonica/
# Dockerfile의 ENV 변수 사용. 이 버전을 AWS AL2에서 빌드할 수 있는지 확인해야 함.
git checkout $LEPTONICA_VERSION
./autogen.sh
# /usr/local에 설치하여 Tesseract가 찾을 수 있도록 함
./configure --prefix=/usr/local
make -j$(nproc) # 사용 가능한 모든 CPU 코어를 사용하여 빌드 속도 향상
make install
echo "--- Leptonica 빌드 및 설치 완료 ---"

# Tesseract 빌드
cd /tmp # 다시 /tmp로 이동하여 Tesseract 소스 다운로드
echo "--- Tesseract 빌드 시작 ---"
git clone https://github.com/tesseract-ocr/tesseract.git
cd tesseract
# Dockerfile의 ENV 변수 사용. 이 버전을 AWS AL2에서 빌드할 수 있는지 확인해야 함.
git checkout $TESSERACT_VERSION
export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig # Leptonica 라이브러리 경로 지정
./autogen.sh
./configure --prefix=/usr/local # /usr/local에 설치
make -j$(nproc)
make install
echo "--- Tesseract 빌드 및 설치 완료 ---"

# Tesseract 언어 데이터 다운로드
echo "--- Tesseract 한국어 학습 데이터(kor.traineddata) 다운로드 ---"
mkdir -p /usr/local/share/tessdata
wget https://github.com/tesseract-ocr/tessdata_fast/raw/main/kor.traineddata -P /usr/local/share/tessdata/
echo "--- 학습 데이터 다운로드 완료 ---"

# 불필요한 디버그 심볼 제거 (크기 최적화)
echo "--- 바이너리 및 라이브러리 스트리핑(Strip) 시작 ---"
# /usr/local/bin 및 /usr/local/lib 에 설치된 바이너리/라이브러리에서 디버그 심볼 제거
# 오류가 발생해도 스크립트가 중단되지 않도록 -f 옵션 사용
find /usr/local/bin -type f -exec strip --strip-all {} + || true
find /usr/local/lib -type f -name "*.so*" -exec strip --strip-all {} + || true
echo "--- 스트리핑 완료 ---"

# 최종적으로 필요한 Tesseract 파일들을 컨테이너의 /opt/ 디렉토리로 복사
# Lambda 런타임은 /opt/ 디렉토리에 있는 파일들을 자동으로 로드합니다.
echo "--- Tesseract 파일들을 /opt/ 디렉토리로 복사 ---"
mkdir -p /opt/bin
mkdir -p /opt/lib
mkdir -p /opt/share/tessdata

# Tesseract 바이너리 복사
cp /usr/local/bin/tesseract /opt/bin/tesseract

# Tesseract 및 Leptonica 라이브러리 복사
cp /usr/local/lib/libleptonica* /opt/lib/
cp /usr/local/lib/libtesseract* /opt/lib/

# Tesseract 5.x의 핵심 의존성: ICU 라이브러리 복사
# 이들은 Lambda 런타임에 없을 가능성이 높으므로 명시적으로 포함합니다.
# ldd 결과에 따라 추가된 다른 라이브러리도 이곳에 포함되어야 함.
cp /usr/local/lib/libicuuc* /opt/lib/ || true
cp /usr/local/lib/libicui18n* /opt/lib/ || true
cp /usr/local/lib/libicudata* /opt/lib/ || true
cp /usr/local/lib/libicuio* /opt/lib/ || true
cp /usr/local/lib/libicule* /opt/lib/ || true
cp /usr/local/lib/libicutu* /opt/lib/ || true

# ldd 결과에서 'not found'로 나온 라이브러리 중
# sh5080/tesseract 이미지에 있었지만 AL2에는 없는 것들을 복사합니다.
# 이 라이브러리들은 Tesseract 빌드 과정에서 /usr/local/lib 에 설치되었을 것입니다.
cp /usr/local/lib/libarchive.so* /opt/lib/ || true # 추가됨
cp /usr/local/lib/libzstd.so* /opt/lib/ || true   # 추가됨
# libwebpmux, libdeflate 등은 Tesseract가 실제로 필요로 하는지에 따라 추가 (ldd 결과에 따라)
# 만약 위의 yum install에서 '-devel' 패키지를 설치했다면,
# 해당 라이브러리는 빌드 시스템에서 사용 가능하므로, 여기서 명시적으로 복사하지 않아도 될 수 있습니다.
# 하지만 AL2의 기본 라이브러리 버전이 Tesseract 빌드 버전과 다를 경우 충돌이 생길 수 있으므로,
# /usr/local/lib 에 설치된 Tesseract-specific 버전을 복사하는 것이 더 안전합니다.

# Tesseract 학습 데이터 복사
cp -r /usr/local/share/tessdata/* /opt/share/tessdata/

echo "--- 파일 /opt/ 복사 완료 ---"

# 복사된 파일들에 실행 권한 설정
chmod -R 755 /opt/bin /opt/lib /opt/share

echo "--- Tesseract 빌드 스크립트 실행 완료 ---"
