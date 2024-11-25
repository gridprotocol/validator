#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <cuda.h>
#include "sha256.cuh"
#include <dirent.h>
#include <ctype.h>
#include <sys/time.h>
// #include <openssl/sha.h>
#include <time.h>

__global__ void sha256_cuda(int diffcult, int index, BYTE *data, size_t len, long long *result, long long n)
{
    long long i = blockIdx.x * blockDim.x + threadIdx.x;
    long long start = index * n;
    // perform sha256 calculation here
    if (i < n)
    {
        long long offset = start + i;
        BYTE idx[8];
        idx[0] = (BYTE)(offset & 0xff);
        idx[1] = (BYTE)((offset >> 8) & 0xff);
        idx[2] = (BYTE)((offset >> 16) & 0xff);
        idx[3] = (BYTE)((offset >> 24) & 0xff);
        idx[4] = (BYTE)((offset >> 32) & 0xff);
        idx[5] = (BYTE)((offset >> 40) & 0xff);
        idx[6] = (BYTE)((offset >> 48) & 0xff);
        idx[7] = (BYTE)((offset >> 56) & 0xff);

        SHA256_CTX ctx;
        BYTE hash[32];
        sha256_init(&ctx);
        sha256_update(&ctx, data, len);
        sha256_update(&ctx, idx, 8);
        sha256_final(&ctx, hash);

        if (checkOutput(hash, diffcult) == 0)
        {
            *result = offset;
        }
    }
}

void pre_sha256()
{
    // compy symbols
    checkCudaErrors(cudaMemcpyToSymbol(dev_k, host_k, sizeof(host_k), 0, cudaMemcpyHostToDevice));
}

void runSha256(int diffcult, int index, BYTE *data, size_t len, long long *result, long long n)
{
    int blockSize = 16;
    int numBlocks = (n + blockSize - 1) / blockSize;

    sha256_cuda<<<numBlocks, blockSize>>>(diffcult, index, data, len, result, n);
}

void byteToHexStr(const unsigned char *source, char *dest, int sourceLen)
{
    short i;
    unsigned char highByte, lowByte;

    for (i = 0; i < sourceLen; i++)
    {
        highByte = source[i] >> 4;
        lowByte = source[i] & 0x0f;

        highByte += 0x30;

        if (highByte > 0x39)
            dest[i * 2] = highByte + 0x07;
        else
            dest[i * 2] = highByte;

        lowByte += 0x30;
        if (lowByte > 0x39)
            dest[i * 2 + 1] = lowByte + 0x07;
        else
            dest[i * 2 + 1] = lowByte;
    }
    return;
}

void printOutput(const unsigned char *output, int len)
{
    char *outputHex = (char *)malloc((2 * len + 1) * sizeof(char));
    byteToHexStr(output, outputHex, len);
    outputHex[2 * len] = '\0';

    printf("%s\n", outputHex);
    free(outputHex);

    return;
}

// long long stringToLong(const char *arr)
// {
//     long long res = 0;
//     char sign;
//     int index = 0;

//     if (arr[index] == '-' || arr[index] == '+')
//     {
//         sign = arr[index++];
//     }

//     char c = arr[index++];
//     while (isdigit(c))
//     {
//         res = res * 10 + (c - '0');
//         c = arr[index++];
//     }

//     if (sign == '-')
//     {
//         return -res;
//     }

//     return res;
// }

extern "C"
{
    int generatePOW(char *rand, int len, int diffcult, long long *index)
    {
        BYTE **inputs;
        // BYTE **outputs;
        long long **indexes;
        long long count = 2 << (diffcult + 1);

        int deviceCount;
        cudaGetDeviceCount(&deviceCount);

        inputs = (BYTE **)malloc(deviceCount * sizeof(BYTE *));
        // outputs = (BYTE **)malloc(deviceCount * sizeof(BYTE *));
        indexes = (long long **)malloc(deviceCount * sizeof(long long *));

        for (int i = 0; i < deviceCount; i++)
        {
            checkCudaErrors(cudaSetDevice(i));

            checkCudaErrors(cudaMallocManaged(&inputs[i], len * sizeof(BYTE)));
            // checkCudaErrors(cudaMallocManaged(&outputs[i], 32 * sizeof(BYTE)));
            checkCudaErrors(cudaMallocManaged(&indexes[i], sizeof(long long)));
            checkCudaErrors(cudaMemcpy(inputs[i], rand, len, cudaMemcpyHostToDevice));
        }

        for (int i = 0; i < deviceCount; i++)
        {
            checkCudaErrors(cudaSetDevice(i));

            *indexes[i] = -1;
            pre_sha256();

            runSha256(diffcult, i, inputs[i], len, indexes[i], count / deviceCount);
        }

        for (int i = 0; i < deviceCount; i++)
        {
            checkCudaErrors(cudaSetDevice(i));

            cudaDeviceSynchronize();
        }

        // long long proof;
        for (int i = 0; i < deviceCount; i++)
            if (*indexes[i] != -1)
            {
                *index = *indexes[i];
                break;
            }

        for (int i = 0; i < deviceCount; i++)
        {
            checkCudaErrors(cudaSetDevice(i));

            cudaFree(inputs[i]);
            // cudaFree(outputs[i]);
            cudaFree(indexes[i]);
        }
        free(inputs);
        // free(outputs);
        free(indexes);

        return 0;
    }
}