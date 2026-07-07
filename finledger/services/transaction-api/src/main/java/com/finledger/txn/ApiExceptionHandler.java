package com.finledger.txn;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import java.util.Map;

/** Translates domain errors to clean HTTP responses (no stack traces leaked). */
@RestControllerAdvice
public class ApiExceptionHandler {

    @ExceptionHandler(InsufficientFundsException.class)
    public ResponseEntity<Map<String, String>> insufficient(InsufficientFundsException e) {
        return ResponseEntity.status(HttpStatus.CONFLICT)
            .body(Map.of("error", "insufficient_funds", "detail", e.getMessage()));
    }

    @ExceptionHandler(IllegalArgumentException.class)
    public ResponseEntity<Map<String, String>> badRequest(IllegalArgumentException e) {
        return ResponseEntity.badRequest()
            .body(Map.of("error", "invalid_request", "detail", e.getMessage()));
    }
}
