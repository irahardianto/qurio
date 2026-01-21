from config import Settings
import pytest
import os

def test_settings_load_from_env(monkeypatch):
    # Arrange
    monkeypatch.setenv("RETRY_MAX_ATTEMPTS", "10")
    monkeypatch.setenv("RETRY_INITIAL_DELAY_MS", "500")
    monkeypatch.setenv("RETRY_MAX_DELAY_MS", "30000")
    monkeypatch.setenv("RETRY_BACKOFF_MULTIPLIER", "3")

    # Act
    settings = Settings()

    # Assert
    assert settings.retry_max_attempts == 10
    assert settings.retry_initial_delay_ms == 500
    assert settings.retry_max_delay_ms == 30000
    assert settings.retry_backoff_multiplier == 3

def test_settings_defaults(monkeypatch):
    # Arrange: Ensure strict isolation by removing vars if they exist
    monkeypatch.delenv("RETRY_MAX_ATTEMPTS", raising=False)
    monkeypatch.delenv("RETRY_INITIAL_DELAY_MS", raising=False)
    monkeypatch.delenv("RETRY_MAX_DELAY_MS", raising=False)
    monkeypatch.delenv("RETRY_BACKOFF_MULTIPLIER", raising=False)

    # Act
    settings = Settings()

    # Assert: Should match defaults in config.py
    assert settings.retry_max_attempts == 3
    assert settings.retry_initial_delay_ms == 1000
    assert settings.retry_max_delay_ms == 60000
    assert settings.retry_backoff_multiplier == 2
