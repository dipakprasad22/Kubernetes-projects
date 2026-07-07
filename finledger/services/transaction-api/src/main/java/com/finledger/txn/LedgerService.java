package com.finledger.txn;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.math.BigDecimal;

/**
 * Core ledger logic. The transfer is @Transactional so the debit, the credit,
 * and the transfer record all commit together or not at all — money can never
 * be partially moved. Insufficient funds aborts the whole transaction.
 */
@Service
public class LedgerService {

    private final TransferRepository transfers;
    private final LedgerEntryRepository ledger;

    public LedgerService(TransferRepository transfers, LedgerEntryRepository ledger) {
        this.transfers = transfers;
        this.ledger = ledger;
    }

    @Transactional
    public Transfer transfer(Long from, Long to, BigDecimal amount) {
        if (amount == null || amount.signum() <= 0) {
            throw new IllegalArgumentException("amount must be positive");
        }
        if (from.equals(to)) {
            throw new IllegalArgumentException("from and to accounts must differ");
        }
        // Check sufficient funds (allow overdraft only for a designated funding account id 0).
        BigDecimal balance = ledger.balanceOf(from);
        if (from != 0L && balance.compareTo(amount) < 0) {
            throw new InsufficientFundsException(
                "account " + from + " balance " + balance + " < " + amount);
        }

        Transfer t = new Transfer();
        t.setFromAccount(from);
        t.setToAccount(to);
        t.setAmount(amount);
        t = transfers.save(t);

        // Double-entry: debit source (negative), credit destination (positive).
        ledger.save(new LedgerEntry(from, t.getId(), amount.negate()));
        ledger.save(new LedgerEntry(to, t.getId(), amount));
        // If anything above throws, the whole @Transactional rolls back.
        return t;
    }

    public BigDecimal balance(Long account) {
        return ledger.balanceOf(account);
    }
}
