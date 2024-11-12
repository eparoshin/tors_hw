#include "discovery.h"

#include "request_sender.h"

#include <arpa/inet.h>

#include <numeric>
#include <ranges>
#include <cassert>
#include <iostream>

namespace NClient {
    namespace rv = std::ranges::views;
    namespace {
        constexpr auto InetToHost(std::integral auto i) {
            static_assert(std::endian::native == std::endian::big || std::endian::native == std::endian::little);
            if constexpr (std::endian::native == std::endian::big) {
                return i;
            } else {
                return std::byteswap(i);
            }
        }

        template <typename T>
        std::vector<std::span<const T>> Convert(const std::vector<std::vector<T>>& elems) {
            return std::vector<std::span<const T>>(elems.begin(), elems.end());
        }

        std::vector<std::vector<char>> PrepareRequests(std::span<const TPoint> points, size_t numNodes) {
            assert(points.size() >= 2);
            size_t reqSize = std::max(2ul, (points.size() + numNodes - 1) / numNodes);
            std::vector<std::vector<char>> result;
            while (!points.empty()) {
                size_t numPoints = reqSize + 1 < points.size() ? reqSize : points.size();
                std::vector<char> request;
                size_t sz = InetToHost(numPoints);
                request.insert(request.end(), reinterpret_cast<char*>(&sz), reinterpret_cast<char*>(&sz) + sizeof(sz));
                for (const auto& point : points.first(numPoints)) {
                    request.insert(request.end(), reinterpret_cast<const char*>(&point.x), reinterpret_cast<const char*>(&point.x) + sizeof(point.x));
                    request.insert(request.end(), reinterpret_cast<const char*>(&point.y), reinterpret_cast<const char*>(&point.y) + sizeof(point.y));
                }

                points = points.last(points.size() - numPoints);
                result.emplace_back(std::move(request));
            }

            return result;
        }

    }

    TDiscovery::TDiscovery(uint16_t port) {
    }

    void TDiscovery::UpdateList() {
        std::vector<sockaddr_in> socks;

        for (int i = 12345; i < 12345 + 5; ++i) {
            sockaddr_in addr;
            addr.sin_family = AF_INET;
            addr.sin_port = htons(i);
            {
                auto res = inet_pton(AF_INET, "127.0.0.1", &addr.sin_addr);
                assert(res > 0);
            }

            socks.push_back(addr);

        }

        addrsRef_.Set(std::move(socks));

    }

    double TDiscovery::Calc(std::span<const TPoint> points) const {
        std::cout << "Calc for points size: " << points.size() << std::endl;
        auto addrs = addrsRef_.Aquire();
        assert(addrs);

        auto requests = PrepareRequests(points, addrs->size());
        auto rSp = Convert(requests);
        auto results = TRequestSender(*addrs).SendRequests(rSp);
        auto doubles = results | rv::transform([](std::span<const char> data) {
            assert(data.size() == sizeof(double));
            return *reinterpret_cast<double const *>(data.data());
        });

        return std::accumulate(doubles.begin(), doubles.end(), 0.0);
    }
}
