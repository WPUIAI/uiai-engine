import{describe,expect,it}from"vitest";import{createPairingPresentation}from"../../src/lib/pairing/pairing-presentation";
const room={schema:"focusa.cockpit_pairing_room.v1",room_id:"room_1234567890",nonce:"nonce_1234567890",client_type:"cockpit",daemon_url:"https://focusa.example",status:"awaiting_operator",created_at:"2026-08-03T00:00:00Z",expires_at:"2026-08-03T00:05:00Z",bridge_owner:"cockpit",pair_url:"https://focusa.example/pair/room_1234567890",pair_code:"focus-1234-5678"};
describe("pairing presentation",()=>{
 it("creates an exact same-origin code and QR payload without mutation",()=>{expect(createPairingPresentation(room)).toEqual({schema:"focusa.pairing_presentation.v1",room_id:"room_1234567890",display_code:"FOCUS-1234-5678",pair_href:"https://focusa.example/pair/room_1234567890",qr_payload:"https://focusa.example/pair/room_1234567890",expires_at:"2026-08-03T00:05:00Z"});});
 it.each([{...room,pair_url:"https://evil.example/pair/x"},{...room,pair_url:"https://focusa.example/pair?token=no"},{...room,pair_code:"bad code"},{...room,status:"completed"}])("rejects unsafe presentation %#",value=>expect(()=>createPairingPresentation(value)).toThrow());
});
