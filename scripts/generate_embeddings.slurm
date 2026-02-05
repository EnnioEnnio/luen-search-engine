#!/bin/bash
#SBATCH -A {account_name}
#SBATCH --mail-type=ALL
#SBATCH --mail-user=slack:<YOUR_USERNAME>
#SBATCH --partition=gpu-batch
#SBATCH -C ARCH:X86
#SBATCH --time=6:00:00
#SBATCH --gpus=1
#SBATCH --mem=64G
#SBATCH --job-name=luen-embeddings
#SBATCH --output=slurm-embeddings-%j.out
#SBATCH --exclude=gx21,gp[01-04]

# This script generates embeddings for the MS MARCO dataset using GPU
# IMPORTANT: Update REPO_PATH to match your cluster home directory path
# NOTE: Excludes V100 GPUs (CUDA 7.0) as PyTorch 25.01 requires CUDA capability >= 7.5

REPO_PATH=/sc/home/<YOUR_USERNAME>/luen-search-engine

echo "Starting embedding generation job"
echo "Job ID: $SLURM_JOB_ID"
echo "Node: $SLURMD_NODENAME"
echo "GPU: $CUDA_VISIBLE_DEVICES"
nvidia-smi --query-gpu=name,compute_cap --format=csv,noheader

# Step 1: Set up container with dependencies (reuses container across jobs)
echo "Setting up container with dependencies..."
CONTAINER_NAME=luen-embed-$USER-pytorch2501
srun --container-image="nvcr.io#nvidia/pytorch:25.01-py3" \
     --container-name=$CONTAINER_NAME \
     --container-writable \
     --container-mounts=$REPO_PATH:/workspace \
     bash -c "pip list | grep -q 'transformers.*5\.' || pip install --no-cache-dir transformers tqdm einops"

# Step 2: Run embedding generation
echo "Running embedding generation..."
srun --container-name=$CONTAINER_NAME \
     --container-mounts=$REPO_PATH:/workspace \
     --container-workdir=/workspace/embedding \
     python create_embedding.py \
       --data /workspace/data/msmarco-docs-preprocessed.tsv \
       --output-dir /workspace/embedding/output \
       --max-length 2048 \
       --batch-size 64 \
       --dim 64

echo "Job completed!"
echo "Output files:"
echo "  - embedding/output/embeddings.npy"
echo "  - embedding/output/doc_ids.npy"
