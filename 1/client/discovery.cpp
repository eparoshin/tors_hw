#include "discovery.h"

#include "request_sender.h"

#include <arpa/inet.h>
#include <sys/poll.h>

#include <cstring>
#include <unordered_set>
#include <numeric>
#include <ranges>
#include <cassert>
#include <string_view>
#include <iostream>
#include <chrono>
#include <array>

namespace std {
    template <>
    struct hash<sockaddr_in> {
        uint64_t operator()(const sockaddr_in& addr) const {
            std::string_view addrData(reinterpret_cast<const char*>(&addr), reinterpret_cast<const char*>(&addr.sin_zero));
            return std::hash<std::string_view>{}(addrData);
        }
    };


}

static bool operator==(const sockaddr_in& a, const sockaddr_in& b) {
        std::string_view aData(reinterpret_cast<const char*>(&a), reinterpret_cast<const char*>(&a.sin_zero));
        std::string_view bData(reinterpret_cast<const char*>(&b), reinterpret_cast<const char*>(&b.sin_zero));

        return aData == bData;
}

namespace NClient {
    namespace rv = std::ranges::views;
    namespace {
        constexpr size_t kBuffSize = 1024;
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

        std::vector<sockaddr_in> BroadcastAndListen(int sockfd, in_addr broadcastAddr, uint16_t port, int timeout) {
            sockaddr_in addr;
            std::memset(&addr, 0, sizeof(addr));
            addr.sin_family = AF_INET;
            addr.sin_port = htons(port);
            addr.sin_addr = broadcastAddr;

            constexpr std::string_view req("PING");

            if (sendto(sockfd, req.data(), req.size(), 0, (sockaddr*)&addr, sizeof(addr)) < 0) {
                throw std::runtime_error("Failed to send ping");
            }

            pollfd fd;


            std::unordered_set<sockaddr_in> addrs;
            auto startTime = std::chrono::steady_clock::now();
            while (1) {
                auto ret = poll(&fd, 1, timeout);
                if (ret < 0) {
                    throw std::runtime_error("Poll failed");
                } else if (ret == 0) {
                    break;
                } else {
                    while (1) {
                        std::array<char, kBuffSize> buff;
                        sockaddr_in recvAddr;
                        socklen_t addrLen = sizeof(recvAddr);
                        ret = recvfrom(sockfd, buff.data(), buff.size(), MSG_DONTWAIT, (struct sockaddr *)&recvAddr, &addrLen);
                        if (ret < 0) {
                            if (errno == EAGAIN || errno == EWOULDBLOCK) {
                                // No more data to read at this moment
                                break;
                            } else {
                                throw std::runtime_error("Error in recvfrom");
                            }
                        } else {
                            sockaddr_in addr;
                            addr.sin_family = AF_INET;
                            addr.sin_port = htons(port);
                            addr.sin_addr = recvAddr.sin_addr;
                            addrs.insert(addr);
                        }
                    }

                    auto currTime = std::chrono::steady_clock::now();
                    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(currTime - startTime).count();
                    timeout -= elapsed;
                    if (timeout <= 0) {
                        break;
                    }
                }
            }

            return std::vector<sockaddr_in>(addrs.begin(), addrs.end());

        }
    }

    TDiscovery::TDiscovery(in_addr broadcastAddr, uint16_t port)
    : broadcastAddr_(broadcastAddr)
    , port_(port)
    , sockFd_(socket(AF_INET, SOCK_DGRAM, 0)) {
        if (sockFd_ < 0) {
            throw std::runtime_error("Socket creation failed");
        }


        int broadcastEnable = 1;
        if (setsockopt(sockFd_, SOL_SOCKET, SO_BROADCAST, &broadcastEnable, sizeof(broadcastEnable)) < 0) {
            close(sockFd_);
            throw std::runtime_error("set broadcast failed");
        }

        sockaddr_in addr;
        std::memset(&addr, 0, sizeof(addr));
        addr.sin_family = AF_INET;
        addr.sin_port = htons(port_);
        addr.sin_addr.s_addr = htonl(INADDR_ANY);

        if (bind(sockFd_, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
            close(sockFd_);
            throw std::runtime_error("bind failed");
        }
    }

    TDiscovery::~TDiscovery() {
        close(sockFd_);
    }

    void TDiscovery::UpdateList(int timeout) {
        addrsRef_.Set(BroadcastAndListen(sockFd_, broadcastAddr_, port_, timeout));
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
