package com.finledger.txn;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import java.math.BigDecimal;

public interface LedgerEntryRepository extends JpaRepository<LedgerEntry, Long> {

    // An account's balance is the sum of its signed ledger entries.
    @Query("SELECT COALESCE(SUM(e.amount), 0) FROM LedgerEntry e WHERE e.accountId = :acct")
    BigDecimal balanceOf(@Param("acct") Long accountId);

    // The whole ledger must sum to zero (used by reconciliation too).
    @Query("SELECT COALESCE(SUM(e.amount), 0) FROM LedgerEntry e")
    BigDecimal totalSum();
}
