# HPC Cluster Instructions for Embedding Generation

This guide explains how to generate embeddings for the MS MARCO dataset on the HPC cluster using GPU resources.

## Prerequisites

1. Clone this repository to your HPC cluster home directory:
   ```bash
   cd /sc/home/<YOUR_USERNAME>
   git clone <repository-url> luen-search-engine
   ```

2. Copy the preprocessed dataset to the cluster:
   ```bash
   # On your local machine:
   scp data/msmarco-docs-preprocessed.tsv user@cluster:/sc/home/<YOUR_USERNAME>/luen-search-engine/data/
   ```

3. Copy the index directory to the cluster (if needed for other operations):
   ```bash
   # On your local machine:
   scp -r index/ user@cluster:/sc/home/<YOUR_USERNAME>/luen-search-engine/
   ```

## Running the Embedding Generation

### Configure the Script

Before first use, edit the script to set your repository path and notification preferences:

```bash
# Edit the script
nano scripts/generate_embeddings.slurm

# Update these lines:
REPO_PATH=/sc/home/<YOUR_USERNAME>/luen-search-engine
#SBATCH --mail-user=<YOUR_EMAIL_OR_SLACK>  # e.g., your.name@hpi.de or slack:username
```

### Submit the Job

From your HPC cluster, navigate to the repository and submit the job:

```bash
cd /sc/home/<YOUR_USERNAME>/luen-search-engine
sbatch scripts/generate_embeddings.slurm
```

This will output something like:
```
Submitted batch job 5698
```

### Monitor the Job

Check job status:
```bash
squeue -u $USER
```

View the output log in real-time:
```bash
tail -f slurm-embeddings-<jobid>.out
```

Example: `tail -f slurm-embeddings-5698.out`

### Cancel the Job (if needed)

```bash
scancel <jobid>
```

## What the Script Does

1. **Allocates resources:**
   - 1 GPU (x86 architecture)
   - 32GB RAM
   - 6 hours time limit

2. **Sets up container (first run only):**
   - Downloads NVIDIA PyTorch container (~5GB)
   - Creates a writable container named `luen-embed-<username>`
   - Installs Python dependencies: transformers, tqdm, einops

3. **Generates embeddings:**
   - Processes all 3.2M documents from `data/msmarco-docs-preprocessed.tsv`
   - Uses `--max-length=2048` for higher quality embeddings
   - Outputs to `embedding/output/` directory
   - Creates two files:
     - `embeddings.npy` (~200MB for dimension 64)
     - `doc_ids.npy` (~13MB)

## Expected Runtime

- Approximately 4-6 hours with a single GPU (A100/V100)
- Progress is tracked with tqdm and visible in the SLURM output log

## Verifying Results

After the job completes successfully:

1. Check the output files:
   ```bash
   ls -lh embedding/output/
   ```

2. Copy the files back to your local machine:
   ```bash
   # On your local machine:
   scp -r user@cluster:/sc/home/<YOUR_USERNAME>/luen-search-engine/embedding/output/ ./embedding/
   ```

3. Verify the embeddings in Python:
   ```python
   import numpy as np
   embeddings = np.load("embedding/output/embeddings.npy")
   doc_ids = np.load("embedding/output/doc_ids.npy")
   print(f"Embeddings shape: {embeddings.shape}")
   print(f"Number of documents: {len(doc_ids)}")
   # Expected: Shape (3213835, 64), Docs: 3213835
   ```

## Troubleshooting

### Job fails with out-of-memory error
Reduce the batch size by editing the script:
```bash
# Change line 35:
--batch-size 32  # or 16 for even less memory usage
```

### Container download is slow
This is normal for the first run (~5GB download). Subsequent runs will reuse the cached container.

### Model download fails
The script downloads the `nomic-ai/nomic-embed-text-v1.5` model (~500MB) from HuggingFace. If this fails:
- Check cluster internet connectivity
- Contact HPC support if HuggingFace is blocked

### GPU not detected
Add this line before the python command in the script to debug:
```bash
nvidia-smi
```

### Monitor GPU usage during execution
SSH to the compute node and run:
```bash
# Find the node name from slurm output
ssh <nodename>
watch -n 1 nvidia-smi
```

## Container Management

The script creates a persistent container named `luen-embed-<username>`. To remove it and start fresh:

```bash
# Check existing containers
enroot list

# Remove the container
enroot remove luen-embed-$USER
```

## Notes

- The container persists between runs, so dependencies only need to be installed once
- The HuggingFace model cache is also preserved in the container
- Output files are written directly to your mounted repository directory
- Make sure you have enough disk space in your home directory (~1GB for all outputs)
