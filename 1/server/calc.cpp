#include "calc.h"

#include <algorithm>
#include <numeric>
#include <ranges>

namespace NServer {
    double CalculateSquare(std::span<const TPoint> points) {
        namespace rv = std::ranges::views;
        auto rng = points
            | rv::slide(2)
            | rv::transform([](auto&& points) { return (points[1].x - points[0].x) * points[0].y; });
        return std::accumulate(rng.begin(), rng.end(), 0.0);
    }
}
