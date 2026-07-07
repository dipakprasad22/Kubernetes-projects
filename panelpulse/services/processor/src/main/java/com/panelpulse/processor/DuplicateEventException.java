package com.panelpulse.processor;
public class DuplicateEventException extends RuntimeException {
    public DuplicateEventException(String key) { super("duplicate: " + key); }
}
