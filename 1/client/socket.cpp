#include "socket.h"

#include <sys/socket.h>
#include <cerrno>
#include <unistd.h>
#include <fcntl.h>
#include <sys/poll.h>

#include <cassert>
#include <array>
#include <stdexcept>
#include <iostream>

namespace NUtil {
    namespace {
        constexpr size_t kBuffSize = 1024;
    }
    int TSocketSet::RegisterSocket(sockaddr_in addr, std::span<const char> request) {
        TSocket sock{addr, request};
        sock.Fd = socket(AF_INET, SOCK_STREAM, 0);
        if (sock.Fd < 0) {
            throw std::runtime_error("Socket creation failed");
        }

        timeval timeout;
        timeout.tv_sec = 1;
        timeout.tv_usec = 0;

        if (setsockopt(sock.Fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout)) < 0) {
            throw std::runtime_error("Failed to set receive timeout");
        }

        if (setsockopt(sock.Fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout)) < 0) {
            throw std::runtime_error("Failed to set send timeout");
        }

        //non blocking
        int flags = fcntl(sock.Fd, F_GETFL, 0);
        fcntl(sock.Fd, F_SETFL, flags | O_NONBLOCK);

        int res = connect(sock.Fd, (struct sockaddr*)&addr, sizeof(addr));
        if (res == 0) {
            sock.State = ESockState::ACTIVE;
        } else if (errno == EINPROGRESS) {
            // connection is in progress
            sock.State = ESockState::CONNECTING;
        } else {
            close(sock.Fd);
            throw std::runtime_error("Socket connection failed");
        }

        sockets_.push_back(sock);
        pollfd pfd;
        pfd.fd = sock.Fd;
        pfd.events = POLLOUT | POLLIN;
        pfds_.push_back(pfd);
        std::cout << "register socket: " << sock.Fd << std::endl;

        return sock.Fd;
    }

    std::optional<std::vector<TSocket>> TSocketSet::Poll() {
        assert(sockets_.size() == pfds_.size());
        if (sockets_.empty()) {
            return std::nullopt;
        }

        std::vector<size_t> idxsToDelete;
        int ret = poll(pfds_.data(), pfds_.size(), -1);
        if (ret < 0) {
            throw std::runtime_error("Poll failed");
        } else if (ret == 0) {
            //timeout
            assert(false);
        } else {
            //success
            for (size_t i = 0; i < sockets_.size(); ++i) {
                auto& pfd = pfds_[i];
                auto& sock = sockets_[i];
                assert(pfd.fd == sock.Fd);

                if (pfd.revents & POLLERR) {
                    sock.State = ESockState::ERROR;
                    idxsToDelete.push_back(i);
                    continue;
                }

                if (pfd.revents & POLLOUT) {
                    auto n = send(pfd.fd, sock.Request.data(), sock.Request.size(), 0);
                    if (n < 0) {
                        sock.State = ESockState::ERROR;
                        idxsToDelete.push_back(i);
                        continue;
                    } else {
                        sock.Request = sock.Request.last(sock.Request.size() - n);
                    }
                }

                if (pfd.revents & (POLLIN | POLLHUP)) {
                    std::array<char, kBuffSize> buff;
                    auto n = recv(pfd.fd, buff.data(), buff.size(), 0);
                    if (n < 0) {
                        sock.State = ESockState::ERROR;
                        idxsToDelete.push_back(i);
                        continue;
                    } else {
                        sock.Response.insert(sock.Response.end(), buff.begin(), buff.begin() + n);
                        if (n == 0) {
                            sock.State = ESockState::CLOSED;
                            idxsToDelete.push_back(i);
                            continue;
                        }
                    }
                }

                pfd.revents = 0;
            }
        }

        std::vector<TSocket> result;
        result.reserve(idxsToDelete.size());

        int shift = 0;
        for (auto idx : idxsToDelete) {
            idx -= (shift++);
            result.emplace_back(std::move(sockets_[idx]));
            sockets_.erase(sockets_.begin() + idx);
            pfds_.erase(pfds_.begin() + idx);
            close(result.back().Fd);
        }

        return result;
    }
}
