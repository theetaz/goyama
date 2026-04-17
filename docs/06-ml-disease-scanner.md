# 06 — ML: disease & pest scanner

## Product intent

Given a phone photo of a plant part, return the most likely diseases/pests with confidence and actionable remedies — in seconds, online or offline, with a closed feedback loop for continuous improvement.

## Data

### Baseline (pre-training)
- **PlantVillage** — open dataset, 54k+ images across 14 crops and 26 diseases. Useful priors; lab-grade photos, so domain gap to in-field phone photos is real.
- **PlantDoc** — ~2.6k in-field images, noisier, better transfer.
- **iNaturalist** (Plantae + Insecta) — wild conditions; good for confusability negatives.
- **Kaggle plant disease datasets** — mixed quality; curate.

### Sri Lanka-specific (the moat)
This is where the product wins. Start collecting from day one via three streams:
1. **Curated field shoots** — partner with HORDI, RRDI, FCRDI field stations. Paid photographer rounds in Wet/Intermediate/Dry zones for priority crops × common diseases, 100–300 images per disease-stage-crop triplet. Budget for ~6 months.
2. **Agronomist contributions** — field officers upload through the CMS during extension visits.
3. **User submissions** — scans with user-confirmed labels become labeled data after expert review; unconfirmed ones are weak labels for semi-supervised learning.

Every image stores: crop, disease, stage (early/mid/severe), affected part, GN-division fuzzed geom, capture device type, license.

### Target coverage for v1
- 30 priority crops × 6 diseases (avg) × 3 stages × ≥ 100 images = ~54k Sri Lanka-specific labeled images.
- Expect 3–6 months to reach with a dedicated ops lead.

## Model

### Phase 1 — classification baseline
- Architecture: **EfficientNet-B0** (server) and **EfficientNet-Lite0 / MobileNetV3-Small** (on-device). Quantized INT8 for TFLite / Core ML.
- Input: 224×224 RGB, normalized. Training augmentation: random crop, horizontal flip, color jitter, RandAugment, cutout; ensure no vertical flip for crops where orientation matters.
- Multi-task head: disease class + crop class + affected-part class. Helps with disambiguation.
- Pretrain on PlantVillage + PlantDoc → fine-tune on Sri Lanka set.
- Loss: cross-entropy + focal for long-tail classes.

### Phase 2 — open-set + detection
- Add an **"other / unknown"** bucket with calibrated confidence (temperature scaling + deep ensembles or MC dropout). Refuse low-confidence predictions rather than guessing.
- Move to detection (**YOLOv8-n** or **RT-DETR**) to handle multiple lesions per image and pests (vs. the whole-image classifier for diseases).
- Optional: vision-language model (SigLIP / OpenCLIP) for zero-shot pest identification where labeled data is scarce.

### Phase 3 — multimodal
- Combine image + declared crop + plot context (AEZ, season, recent weather, local disease pressure) for a reranker. An LLM post-processor (small on-server model) can generate user-friendly explanations from structured predictions.

## Serving

### On-device
- TFLite or Core ML model bundled in the app (≤ 20 MB quantized). Inference ≤ 500 ms on a mid-tier Android (e.g., Redmi Note-class).
- Runs offline. Results always written to local DB and queued for upload when connectivity returns.

### Server
- ONNX Runtime behind FastAPI. Used for:
  - Fallback when device model confidence is low.
  - The authoritative model used during review / retraining.
  - Batch re-labeling of the archive when a new model version ships.
- Image preprocessing done once, cached. Grad-CAM overlay generated on request.

## Review & feedback loop

```
scan (user)
   │
   ├── on-device inference ──▶ show top-3 ──▶ user confirms / disagrees
   │                                              │
   │                                              ▼
   └──────────────────────────────────────▶ server store
                                                  │
                                                  ├── low confidence?  ─┐
                                                  ├── user disagreed?   ├──▶ agronomist review queue
                                                  └── random sampling ──┘
                                                                          │
                                                   expert-confirmed label │
                                                                          ▼
                                                                    training corpus
                                                                          │
                                                              weekly fine-tune
                                                                          │
                                                                  candidate model
                                                                          │
                                                                  offline eval
                                                                          │
                                                                  shadow deploy
                                                                          │
                                                                 promote to prod
```

## Evaluation

- **Held-out Sri Lanka test set** — frozen, built from independent capture rounds (not scraped from training sources).
- Metrics: top-1, top-3, macro-F1, per-crop F1, per-stage F1, coverage-at-confidence (what % of scans clear the confidence threshold).
- **Confusion-pair dashboard** — track pairs like powdery-mildew vs. downy-mildew on cucurbits; agronomist-reviewed confusions get targeted data collection.
- **Fairness / robustness** — slice metrics by device brand, lighting class, zone, language of the declaring user.
- Ship a new model only if it improves top-3 on the held-out set by ≥ 1 point AND does not regress any single-crop F1 by > 2 points.

## MLOps

- Model registry — MLflow or a simple S3 bucket with JSON manifest.
- Dataset versioning — DVC or LakeFS pointing at object storage.
- Training — weekly job on a single A10/A100 instance (rent by the hour; Lambda Labs / Modal / Runpod).
- CI: reproducible training image, deterministic splits, metric gate.
- Canary: 5% of server inferences served by candidate; compare disagreement vs. champion for 48 h.

## Safety & ethics

- Never auto-recommend a chemical without agronomist-approved dosage and PHI. The model identifies the disease; the remedy comes from the curated DB.
- Always show confidence; never single-result UI below threshold.
- User consent for contributing scans to training. Opt-out at any time.
- Store training images without PII; fuzz location to GN-division.
- Handle hallucination: if the image is clearly not a plant, return a polite "I don't see a plant" (a small up-front gate classifier).

## Budget rough estimate

- Data collection: LKR 1.5–3M over 6 months (photographer + agronomist time + transport).
- GPU training: ~$200–500/month during active training cycles.
- Serving: CPU-only inference fits a $30–60/month VPS for early scale; GPU only if scan volume justifies.
