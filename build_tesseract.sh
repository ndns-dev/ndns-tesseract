# 오류 발생 시 즉시 스크립트 중단
set -e

# 초기 위치를 /tmp로 설정 (Dockerfile의 WORKDIR과 일치)
cd /tmp

# Leptonica 빌드
echo "--- Leptonica 빌드 시작 ---"
git clone https://github.com/DanBloomberg/leptonica.git
cd leptonica/
git checkout $LEPTONICA_VERSION # Dockerfile의 ENV 변수 사용
./autogen.sh
./configure --prefix=/usr/local # /usr/local에 설치하여 Tesseract가 찾을 수 있도록 함
make -j$(nproc) # 사용 가능한 모든 CPU 코어를 사용하여 빌드 속도 향상
make install
echo "--- Leptonica 빌드 및 설치 완료 ---"

# Tesseract 빌드
cd /tmp # 다시 /tmp로 이동하여 Tesseract 소스 다운로드
echo "--- Tesseract 빌드 시작 ---"
git clone https://github.com/tesseract-ocr/tesseract.git
cd tesseract
git checkout $TESSERACT_VERSION # Dockerfile의 ENV 변수 사용
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
# /usr/local/bin 및 /usr/local/lib 에 설치된 바이너리/라이브러리에서 디버그 심볼 제거
echo "--- 바이너리 및 라이브러리 스트리핑(Strip) 시작 ---"
find /usr/local/bin -type f -exec strip {} \; || true
find /usr/local/lib -type f -name "*.so*" -exec strip {} \; || true
echo "--- 스트리핑 완료 ---"


# 최종적으로 필요한 Tesseract 파일들을 컨테이너의 /opt/ 디렉토리로 복사
echo "--- Tesseract 파일들을 /opt/ 디렉토리로 복사 ---"
mkdir -p /opt/bin
mkdir -p /opt/lib
mkdir -p /opt/share/tessdata

# Tesseract 바이너리 복사
cp /usr/local/bin/tesseract /opt/bin/tesseract || true

# Tesseract 및 Leptonica 라이브러리 복사
cp /usr/local/lib/libleptonica* /opt/lib/ || true
cp /usr/local/lib/libtesseract* /opt/lib/ || true

# Tesseract 5.x의 핵심 의존성: ICU 라이브러리 복사
cp /usr/local/lib/libicuuc* /opt/lib/ || true
cp /usr/local/lib/libicui18n* /opt/lib/ || true
cp /usr/local/lib/libicudata* /opt/lib/ || true
cp /usr/local/lib/libicuio* /opt/lib/ || true
cp /usr/local/lib/libicule* /opt/lib/ || true
cp /usr/local/lib/libicutu* /opt/lib/ || true

# Tesseract 학습 데이터 복사
cp -r /usr/local/share/tessdata/* /opt/share/tessdata/ || true

echo "--- 파일 /opt/ 복사 완료 ---"

# 복사된 파일들에 실행 권한 설정 (선택적이나, 안전을 위해)
chmod -R 755 /opt/bin /opt/lib /opt/share || true

echo "--- Tesseract 빌드 스크립트 실행 완료 ---"