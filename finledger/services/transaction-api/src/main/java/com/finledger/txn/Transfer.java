package com.finledger.txn;

import jakarta.persistence.*;
import java.math.BigDecimal;
import java.time.Instant;

/**
 * A money transfer request. Records the intent and outcome; the actual
 * balance movement lives in two LedgerEntry rows (debit + credit).
 */
@Entity
@Table(name = "transfers")
public class Transfer {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false)
    private Long fromAccount;

    @Column(nullable = false)
    private Long toAccount;

    // BigDecimal for money — never use float/double for currency.
    @Column(nullable = false, precision = 19, scale = 4)
    private BigDecimal amount;

    @Column(nullable = false)
    private String status = "completed";

    @Column(nullable = false)
    private Instant createdAt = Instant.now();

    // getters / setters
    public Long getId() { return id; }
    public Long getFromAccount() { return fromAccount; }
    public void setFromAccount(Long v) { this.fromAccount = v; }
    public Long getToAccount() { return toAccount; }
    public void setToAccount(Long v) { this.toAccount = v; }
    public BigDecimal getAmount() { return amount; }
    public void setAmount(BigDecimal v) { this.amount = v; }
    public String getStatus() { return status; }
    public void setStatus(String v) { this.status = v; }
    public Instant getCreatedAt() { return createdAt; }
}
