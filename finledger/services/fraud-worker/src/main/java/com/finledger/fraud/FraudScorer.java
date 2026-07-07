package com.finledger.fraud;

import java.math.BigDecimal;

/**
 * Simple, explainable fraud scoring. Real systems use ML models / rules engines;
 * this captures the shape: score a transaction, flag if over threshold.
 */
public class FraudScorer {

    private static final BigDecimal LARGE_AMOUNT = new BigDecimal("10000");

    /** Returns a risk score 0..100. */
    public int score(long fromAccount, long toAccount, BigDecimal amount) {
        int risk = 0;
        if (amount.compareTo(LARGE_AMOUNT) > 0) risk += 50;     // large transfer
        if (toAccount > 9_000_000L) risk += 30;                  // flagged account range
        if (amount.scale() > 2) risk += 10;                      // odd precision
        return Math.min(risk, 100);
    }

    public boolean isSuspicious(long from, long to, BigDecimal amount) {
        return score(from, to, amount) >= 60;
    }
}
