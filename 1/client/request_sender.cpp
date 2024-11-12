#include "request_sender.h"

#include "socket.h"

#include <queue>
#include <utility>
#include <algorithm>
#include <cassert>
#include <ranges>
#include <unordered_map>

namespace NClient {
    namespace {
        enum class EReqState {
            FINISHED,
            FD_ASSIGNED,
            FD_NOT_ASSIGNED,
        };

        struct TRequest {
            TRequestSender::TEndpoint* Endpoint;
            std::span<const char> Request;
            std::vector<char> Response;
            EReqState State = EReqState::FD_NOT_ASSIGNED;
        };
    }
    namespace rv = std::ranges::views;
    TRequestSender::TRequestSender(std::span<const sockaddr_in> endpoints) {
        endpoints_.resize(endpoints.size());
        std::transform(endpoints.begin(), endpoints.end(), endpoints_.begin(), [](auto addr) {return TEndpoint{addr}; });
    }

    TRequestSender::TEndpoint& TRequestSender::NextEndpoint() {
        for (int i = curr_idx_; i < curr_idx_ + endpoints_.size(); ++i) {
            if (endpoints_[i % endpoints_.size()].Alive) {
                curr_idx_ = i + 1;
                return endpoints_[i % endpoints_.size()];
            }
        }

        throw std::runtime_error("No living endpoints left");
    }

    std::vector<std::vector<char>> TRequestSender::SendRequests(std::span<const std::span<const char>> requests) && {

        std::vector<TRequest> reqs;
        std::transform(requests.begin(), requests.end(), std::back_inserter(reqs), [this, requests, idx = 0](auto req) mutable {
            return TRequest{&NextEndpoint(), requests[idx++]};
        });


        NUtil::TSocketSet socketSet;
        std::unordered_map<int, size_t> fdIdxs;
        size_t num_finished = 0;
        while (num_finished < reqs.size()) {
            for (size_t i = 0; i < reqs.size(); ++i) {
                auto& req = reqs[i];
                if (req.State == EReqState::FINISHED || req.State == EReqState::FD_ASSIGNED) {
                    continue;
                }

                auto fd = socketSet.RegisterSocket(req.Endpoint->Addr, req.Request);
                req.State = EReqState::FD_ASSIGNED;
                fdIdxs[fd] = i;
            }

            auto sockets = socketSet.Poll();
            if (sockets) {
                for (auto& sock : *sockets) {
                    auto& req = reqs.at(fdIdxs.at(sock.Fd));
                    if (sock.State == NUtil::ESockState::ERROR) {
                        req.Endpoint->Alive = false;
                        req.Endpoint = &NextEndpoint();
                        req.State = EReqState::FD_NOT_ASSIGNED;
                        fdIdxs.erase(sock.Fd);
                    } else if (sock.State == NUtil::ESockState::CLOSED) {
                        ++num_finished;
                        req.State = EReqState::FINISHED;
                        req.Response = std::move(sock.Response);
                        fdIdxs.erase(sock.Fd);
                    } else {
                        assert(false);
                    }
                }
            } else {
                assert(std::ranges::all_of(reqs, [](const auto& req) {
            return req.State == EReqState::FINISHED; }));
            }
        }


        std::vector<std::vector<char>> result(requests.size());
        std::transform(reqs.begin(), reqs.end(), result.begin(), [](auto& req) {
            assert(req.State == EReqState::FINISHED);
            return std::move(req.Response);
        });
        return result;

    }
}
