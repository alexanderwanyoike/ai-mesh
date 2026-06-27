# ai-mesh

A multi-provider CLI for generating 3D meshes (GLB) from text or images with AI.

Supports [Fal](https://fal.ai/) (Tencent Hunyuan3D 3.1) and [Meshy](https://www.meshy.ai/) out of the box, with a clean provider interface for adding more. Built to be scripted, batched, and called by agents - one mesh per invocation, the output path on stdout, progress on stderr.

It is the mesh half of a composable pair with [ai-img](https://github.com/alexanderwanyoike/ai-img): `ai-img` turns a prompt into an image, `ai-mesh` turns an image into a mesh.

## Installation

### From release binaries

Download a prebuilt binary from the [Releases](https://github.com/alexanderwanyoike/ai-mesh/releases) page (Linux, macOS, Windows; amd64 and arm64).

### From source

```bash
go install github.com/alexanderwanyoike/ai-mesh@latest
```

### Build locally

```bash
git clone https://github.com/alexanderwanyoike/ai-mesh.git
cd ai-mesh
make build
```

## Quick Start

1. Set the API key for the provider you want:

```bash
export FAL_API_KEY="your-key-here"     # for fal
export MESHY_API_KEY="your-key-here"   # for meshy
```

2. Generate a mesh from an image:

```bash
ai-mesh -i character.png -o character.glb
```

3. Generate a mesh from a text prompt (Meshy):

```bash
ai-mesh -p meshy "a low-poly treasure chest" -o chest.glb
```

## Configuration

ai-mesh needs an API key for the provider you use. There are three ways to set
one, resolved in this order of precedence:

1. **`--api-key` flag** (per call, handy for agents):
   ```bash
   ai-mesh -p meshy -k "$MY_KEY" "a chest" -o chest.glb
   ```
2. **Environment variable**:
   ```bash
   export FAL_API_KEY="..."      # for -p fal
   export MESHY_API_KEY="..."    # for -p meshy
   ```
3. **Config file** (persistent, no shell env needed):
   ```bash
   ai-mesh config set fal "your-fal-key"
   ai-mesh config set meshy "your-meshy-key"
   ```

The config file is stored at `~/.ai-mesh/config.json` with `0600` permissions.

| Provider | Env Variable | Get a key |
|----------|-------------|-----------|
| Fal      | `FAL_API_KEY`   | [Fal dashboard](https://fal.ai/dashboard/keys) |
| Meshy    | `MESHY_API_KEY` | [Meshy API](https://www.meshy.ai/api) |

### Managing stored keys

```bash
ai-mesh config set <provider> <key>   # store a key
ai-mesh config get <provider>         # show it (masked)
ai-mesh config list                   # status for every provider
ai-mesh config unset <provider>       # remove a stored key
ai-mesh config path                   # print the config file path
```

## Usage

```
ai-mesh [flags] ["prompt"]
```

Provide a `"prompt"` (text-to-3d) or an `--input` image (image-to-3d). The result
is downloaded and written to the `--output` path; the path is echoed to stdout.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--provider` | `-p` | `fal` | Provider: `fal`, `meshy` |
| `--model` | `-m` | per-provider | Model name/ID or endpoint |
| `--output` | `-o` | `output.glb` | Output file path |
| `--input` | `-i` | | Input image for image-to-3d |
| `--api-key` | `-k` | | API key (overrides env var and config) |
| `--faces` | | `50000` | Target face/polygon count |
| `--pbr` | | `false` | Request PBR material maps |
| `--no-wait` | | `false` | Submit the job, print its ID, and exit (don't block) |
| `--timeout` | | `30` | Minutes to wait for a job before giving up |

## Providers

### Fal (default) - image-to-3d

Tencent Hunyuan3D 3.1, hosted on Fal. Image input only; a bare text prompt returns
a hint to pass an image or use Meshy.

| Model (`-m`) | Description |
|--------------|-------------|
| `fal-ai/hunyuan-3d/v3.1/pro/image-to-3d` (default) | High quality, up to 1.5M faces |
| `fal-ai/hunyuan-3d/v3.1/rapid/image-to-3d` | Faster / cheaper |

### Meshy - text-to-3d and image-to-3d

Image input runs Meshy's single-stage image-to-3d. A text prompt runs the
two-stage text-to-3d (preview geometry, then refine with texture).

| Model (`-m`) | Description |
|--------------|-------------|
| `meshy-5` (default) | Meshy 5 |
| `meshy-6` | Meshy 6 |
| `latest` | Newest available |

## Composing

`ai-mesh` does one thing, so it pipes cleanly into batches, agents, and other tools.

### Text -> image -> mesh (with ai-img)

```bash
ai-img "a stone golem, T-pose, plain background" -o golem.png
ai-mesh -i golem.png -o golem.glb
```

### Batch a folder of images

```bash
for img in refs/*.png; do
  ai-mesh -i "$img" -o "meshes/$(basename "${img%.png}").glb"
done
```

### In a script / agent

stdout is just the output path, so it chains:

```bash
mesh=$(ai-mesh -i hero.png -o hero.glb)
blender --background --python import_glb.py -- "$mesh"
```

## Long jobs: fire-and-forget

Some models (notably Fal's `pro` tier) can spend many minutes in cold-start and
queue before compute even begins. Two things help:

- `--timeout <minutes>` (default 30) bounds the wait, generous enough that a slow
  cold-start won't discard a job the provider is still running (and that you paid
  for).
- `--no-wait` submits the job, prints its **ID**, and exits immediately. Retrieve
  the result whenever it's ready with `fetch` (free - no re-generation):

```bash
id=$(ai-mesh -p fal -i hero.png --no-wait)
# ... do other work ...
ai-mesh fetch fal "$id" -o hero.glb
```

`--no-wait` works for the single-stage paths (Fal, Meshy image-to-3d). Meshy
text-to-3d is two-stage and always runs synchronously.

## Development

```bash
make build     # Build binary
make test      # Run tests
make clean     # Remove binary
make install   # go install
```

## License

MIT
