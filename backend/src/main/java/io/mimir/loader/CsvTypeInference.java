package io.mimir.loader;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.format.DateTimeParseException;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

import io.mimir.metadata.Metadata.SourceSystemInterface;

final class CsvTypeInference {
    private static final int MAX_VARCHAR_LENGTH = 16_383;
    private final SourceSystemInterface sourceInterface;
    private final List<ColumnInference> columns;

    CsvTypeInference(SourceSystemInterface sourceInterface, int columnCount) {
        this.sourceInterface = sourceInterface;
        this.columns = new ArrayList<>(columnCount);
        for (int index = 0; index < columnCount; index++) {
            columns.add(new ColumnInference());
        }
    }

    void accept(List<String> row) {
        for (int index = 0; index < row.size(); index++) {
            columns.get(index).accept(row.get(index), sourceInterface);
        }
    }

    List<InferredColumn> finish() {
        return columns.stream().map(ColumnInference::finish).toList();
    }

    static String normalizeNumber(String value, SourceSystemInterface sourceInterface) {
        NumericValue numeric = numericValue(value, sourceInterface);
        return numeric == null ? null : numeric.normalized();
    }

    private static NumericValue numericValue(String value, SourceSystemInterface sourceInterface) {
        String text = value.strip();
        if (text.isEmpty() || sourceInterface.decimalSeparator().equals(sourceInterface.thousandsSeparator())) {
            return null;
        }

        String decimalSeparator = sourceInterface.decimalSeparator();
        String thousandsSeparator = sourceInterface.thousandsSeparator();
        int decimalIndex = text.indexOf(decimalSeparator);
        if (decimalIndex >= 0 && text.indexOf(decimalSeparator, decimalIndex + 1) >= 0) {
            return null;
        }

        String integerPart = decimalIndex < 0 ? text : text.substring(0, decimalIndex);
        String fractionalPart = decimalIndex < 0 ? "" : text.substring(decimalIndex + 1);
        if (decimalIndex >= 0 && (fractionalPart.isEmpty() || !fractionalPart.chars().allMatch(Character::isDigit))) {
            return null;
        }

        boolean negative = integerPart.startsWith("-");
        boolean positive = integerPart.startsWith("+");
        String unsignedInteger = negative || positive ? integerPart.substring(1) : integerPart;
        if (unsignedInteger.isEmpty()) {
            return null;
        }

        String normalizedInteger;
        if (unsignedInteger.contains(thousandsSeparator)) {
            String[] groups = unsignedInteger.split(Pattern.quote(thousandsSeparator), -1);
            if (groups[0].isEmpty() || groups[0].length() > 3
                    || !groups[0].chars().allMatch(Character::isDigit)) {
                return null;
            }
            for (int index = 1; index < groups.length; index++) {
                if (groups[index].length() != 3 || !groups[index].chars().allMatch(Character::isDigit)) {
                    return null;
                }
            }
            normalizedInteger = String.join("", groups);
        } else {
            if (!unsignedInteger.chars().allMatch(Character::isDigit)) {
                return null;
            }
            normalizedInteger = unsignedInteger;
        }

        boolean leadingZero = normalizedInteger.length() > 1 && normalizedInteger.charAt(0) == '0';
        String normalized = (negative ? "-" : positive ? "+" : "") + normalizedInteger
                + (decimalIndex < 0 ? "" : "." + fractionalPart);
        try {
            return leadingZero ? null : new NumericValue(normalized, new BigDecimal(normalized));
        } catch (NumberFormatException exception) {
            return null;
        }
    }

    record InferredColumn(String datatype, Integer characterMaximumLength, Integer numericPrecision, Integer numericScale) {}

    private record NumericValue(String normalized, BigDecimal value) {}

    private static final class ColumnInference {
        private boolean hasValue;
        private boolean allBoolean = true;
        private boolean allInteger = true;
        private boolean allDecimal = true;
        private boolean allDate = true;
        private boolean allDateTime = true;
        private int maxLength;
        private int maxPrecision;
        private int maxScale;

        void accept(String value, SourceSystemInterface sourceInterface) {
            String trimmed = value.strip();
            if (trimmed.isEmpty()) {
                return;
            }
            hasValue = true;
            maxLength = Math.max(maxLength, value.codePointCount(0, value.length()));

            String lowercase = trimmed.toLowerCase(java.util.Locale.ROOT);
            allBoolean &= lowercase.equals("true") || lowercase.equals("false");

            NumericValue numeric = numericValue(trimmed, sourceInterface);
            boolean integer = numeric != null && !trimmed.contains(sourceInterface.decimalSeparator());
            if (integer) {
                try {
                    Long.parseLong(numeric.normalized());
                } catch (NumberFormatException exception) {
                    integer = false;
                }
            }
            allInteger &= integer;
            allDecimal &= numeric != null;
            if (numeric != null) {
                BigDecimal decimal = numeric.value();
                int scale = Math.max(decimal.scale(), 0);
                maxScale = Math.max(maxScale, scale);
                maxPrecision = Math.max(maxPrecision, Math.max(decimal.precision(), scale));
            }

            try {
                LocalDate.parse(trimmed);
            } catch (DateTimeParseException exception) {
                allDate = false;
            }
            try {
                LocalDateTime.parse(trimmed);
            } catch (DateTimeParseException exception) {
                allDateTime = false;
            }
        }

        InferredColumn finish() {
            if (hasValue && allBoolean) {
                return new InferredColumn("BOOLEAN", null, null, null);
            }
            if (hasValue && allInteger) {
                return new InferredColumn("BIGINT", null, 19, 0);
            }
            if (hasValue && allDecimal && maxPrecision <= 65 && maxScale <= 30) {
                return new InferredColumn(
                        "DECIMAL(" + Math.max(maxPrecision, 1) + "," + maxScale + ")",
                        null,
                        maxPrecision,
                        maxScale);
            }
            if (hasValue && allDate) {
                return new InferredColumn("DATE", null, null, null);
            }
            if (hasValue && allDateTime) {
                return new InferredColumn("DATETIME", null, null, null);
            }
            int length = hasValue ? Math.max(maxLength, 1) : 255;
            return new InferredColumn(
                    length <= MAX_VARCHAR_LENGTH ? "VARCHAR(" + length + ")" : "MEDIUMTEXT",
                    length,
                    null,
                    null);
        }
    }
}
