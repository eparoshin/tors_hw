#pragma once

#include <netinet/in.h>

#include <sys/poll.h>
#include <vector>
#include <span>
#include <optional>

namespace NUtil {
    enum class ESockState {
        ERROR,
        CONNECTING,
        ACTIVE,
        CLOSED,
        INIT
    };

    struct TSocket {
        sockaddr_in Addr;
        std::span<const char> Request;
        std::vector<char> Response;
        int Fd = -1;
        ESockState State = ESockState::INIT;
    };

    class TSocketSet {
    public:
        int RegisterSocket(sockaddr_in addr, std::span<const char> request);

        std::optional<std::vector<TSocket>> Poll();
    private:
        std::vector<TSocket> sockets_;
        std::vector<pollfd> pfds_;
    };
}
