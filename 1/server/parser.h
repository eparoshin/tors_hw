#pragma once

#include <vector>
#include <span>

namespace NServer {
    struct TPoint {
        double x;
        double y;
    };

    std::vector<TPoint> Parse(std::span<const char> input);
}
