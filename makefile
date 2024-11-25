APP_NAME=meeda
LIB_NAME=libpow.so
# GIT_COMMIT=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell TZ=Asia/Shanghai date +'%Y-%m-%d.%H:%M:%S%Z')
BUILD_FLAGS=-ldflags "-X 'github.com/memoio/meeda-node/cmd.BuildFlag=$(BUILD_TIME)'"

all: clean cuda build

clean:
	rm -f ${APP_NAME} ${LIB_NAME}

cuda:
	nvcc --ptxas-options=-v --compiler-options '-fPIC' -o ${LIB_NAME} --shared CudaSha256/pow.cu

build:
	go build $(BUILD_FLAGS) -o ${APP_NAME}

install:
	mv ${APP_NAME} /usr/local/bin
	
.PHONY: all clean cuda build