#pragma once

#include "rcu.h"

#include <netinet/in.h>

#include <cstdint>
#include <vector>
#include <thread>
#include <span>

namespace NClient {
    struct TPoint {
        double x;
        double y;
    };

    class TDiscovery {
    public:
        TDiscovery(const TDiscovery&) = delete;
        TDiscovery& operator=(const TDiscovery&) = delete;

        TDiscovery(in_addr broadcastAddr, uint16_t port, int update_period, int timeout);


        double Calc(std::span<const TPoint> points) const;

        ~TDiscovery();
    private:
        void UpdateList(int timeout);

        std::thread updater_;
        in_addr broadcastAddr_;
        uint16_t port_;
        int sockFd_;
        NUtil::TRcu<std::vector<sockaddr_in>> addrsRef_;
    };
}
