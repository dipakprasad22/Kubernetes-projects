package com.finledger.txn;

import jakarta.persistence.*;
import java.math.BigDecimal;
import java.time.Instant;

/**
 * A single immutable, append-only ledger posting. Every transfer creates two:
 * a debit (negative) on the source account and a credit (positive) on the
 * destination. Across the whole ledger, the sum of all entries must be zero —
 * that invariant is what the reconciliation CronJob verifies.
 */
@Entity
@Table(name = "ledger_entries")
public class LedgerEntry {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false)
    private Long accountId;

    @Column(nullable = false)
    private Long transferId;

    // Signed amount: negative = debit, positive = credit.
    @Column(nullable = false, precision = 19, scale = 4)
    private BigDecimal amount;

    @Column(nullable = false)
    private Instant createdAt = Instant.now();

    public LedgerEntry() {}
    public LedgerEntry(Long accountId, Long transferId, BigDecimal amount) {
        this.accountId = accountId;
        this.transferId = transferId;
        this.amount = amount;
    }

    public Long getId() { return id; }
    public Long getAccountId() { return accountId; }
    public Long getTransferId() { return transferId; }
    public BigDecimal getAmount() { return amount; }
}
