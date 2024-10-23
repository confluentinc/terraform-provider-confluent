package io.confluent.flink.table.modules.remoteudf;

import org.apache.flink.table.functions.ScalarFunction;

import java.util.Arrays;
import java.util.List;
import java.util.stream.IntStream;

/** TShirt sizing function for demo. */
public class TShirtSizingIsSmaller extends ScalarFunction {
    public static final String NAME = "IS_SMALLER";

    private static final List<Size> ORDERED_SIZES =
            Arrays.asList(
                    new Size("X-Small", "XS"),
                    new Size("Small", "S"),
                    new Size("Medium", "M"),
                    new Size("Large", "L"),
                    new Size("X-Large", "XL"),
                    new Size("XX-Large", "XXL"));

    public boolean eval(String shirt1, String shirt2) {
        int size1 = findSize(shirt1);
        int size2 = findSize(shirt2);
        // If either can't be found just say false rather than throw an error
        if (size1 == -1 || size2 == -1) {
            return false;
        }
        return size1 < size2;
    }

    private int findSize(String shirt) {
        return IntStream.range(0, ORDERED_SIZES.size())
                .filter(
                        i -> {
                            Size s = ORDERED_SIZES.get(i);
                            return s.name.equalsIgnoreCase(shirt)
                                    || s.abbreviation.equalsIgnoreCase(shirt);
                        })
                .findFirst()
                .orElse(-1);
    }

    private static class Size {
        private final String name;
        private final String abbreviation;

        public Size(String name, String abbreviation) {
            this.name = name;
            this.abbreviation = abbreviation;
        }
    }
}