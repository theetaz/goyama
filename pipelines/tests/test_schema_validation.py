from __future__ import annotations

import uuid
from datetime import UTC, datetime

from core.extractor import validate_record


def _now() -> str:
    return datetime.now(UTC).isoformat()


def test_minimal_crop_validates() -> None:
    record = {
        "slug": "brinjal",
        "version": 1,
        "status": "draft",
        "scientific_name": "Solanum melongena",
        "names": {"en": "Brinjal"},
        "category": "vegetable",
        "life_cycle": "annual",
    }
    errors = validate_record("crop.json", record)
    assert errors == [], errors


def test_chemical_remedy_requires_chemical_block() -> None:
    record = {
        "slug": "mancozeb-tomato-late-blight",
        "version": 1,
        "status": "draft",
        "type": "chemical",
        "target_disease_slugs": ["tomato-late-blight"],
        "name": {"en": "Mancozeb 80% WP"},
    }
    errors = validate_record("remedy.json", record)
    assert any("chemical" in e for e in errors), errors


def test_well_formed_chemical_remedy_validates() -> None:
    record = {
        "slug": "mancozeb-tomato-late-blight",
        "version": 1,
        "status": "draft",
        "type": "chemical",
        "target_disease_slugs": ["tomato-late-blight"],
        "name": {"en": "Mancozeb 80% WP foliar spray"},
        "chemical": {
            "active_ingredient": "Mancozeb",
            "application_method": "foliar_spray",
            "pre_harvest_interval_days": 7,
        },
    }
    errors = validate_record("remedy.json", record)
    assert errors == [], errors


def test_disease_requires_affected_crops() -> None:
    record = {
        "slug": "tomato-late-blight",
        "version": 1,
        "status": "draft",
        "scientific_name": "Phytophthora infestans",
        "names": {"en": "Late blight"},
        "causal_organism": "fungal",
    }
    errors = validate_record("disease.json", record)
    assert any("affected_crop_slugs" in e for e in errors), errors


def test_unknown_record_id() -> None:
    # sanity: uuid format is accepted
    record = {
        "id": str(uuid.uuid4()),
        "slug": "rice",
        "version": 1,
        "status": "draft",
        "scientific_name": "Oryza sativa",
        "names": {"en": "Rice"},
        "category": "field_crop",
        "life_cycle": "annual",
    }
    errors = validate_record("crop.json", record)
    assert errors == [], errors
