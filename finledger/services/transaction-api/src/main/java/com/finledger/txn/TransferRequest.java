package com.finledger.txn;

import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;
import java.math.BigDecimal;

/** Inbound transfer request with validation. */
public class TransferRequest {
    @NotNull public Long fromAccount;
    @NotNull public Long toAccount;
    @NotNull @Positive public BigDecimal amount;
}
