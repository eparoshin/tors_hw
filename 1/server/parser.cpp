#include "parser.h"

#include <ranges>
#include <cassert>
#include <bit>

namespace NServer {
    namespace {
        constexpr auto InetToHost(std::integral auto i) {
            static_assert(std::endian::native == std::endian::big || std::endian::native == std::endian::little);
            if constexpr (std::endian::native == std::endian::big) {
                return i;
            } else {
                return std::byteswap(i);
            }
        }

        std::pair<std::span<const char>, size_t> ParseLen(std::span<const char> input) {
            assert(input.size() >= sizeof(size_t));
            size_t result = InetToHost(*reinterpret_cast<const size_t*>(input.data()));
            return {input.last(input.size() - sizeof(size_t)), result};
        }

        constexpr size_t kDoubleSize = sizeof(double);
        constexpr size_t kPointSize = 2 * kDoubleSize;

        void AssertSpanSize(std::span<const char> input, size_t len) {
            assert(input.size() == len * kPointSize);
        }
    }

    std::vector<TPoint> Parse(std::span<const char> input) {
        size_t len;
        std::tie(input, len) = ParseLen(input);
        AssertSpanSize(input, len);

        namespace rw = std::ranges::views;
        auto rng =  input
            | rw::chunk(kDoubleSize)
            | rw::transform([](std::span<const char> data) -> double {
                return *reinterpret_cast<const double*>(data.data());
            })
            | rw::chunk(2)
            | rw::transform([](auto&& coords) { return TPoint{coords[0], coords[1]}; });

        return std::vector(rng.begin(), rng.end());

    }
}

