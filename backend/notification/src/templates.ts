// path: backend/notification/src/templates.ts

import { Language, NotificationEventData, RenderedMessage } from './types';

export function formatKES(amountCents?: number): string {
  if (amountCents === undefined || isNaN(amountCents)) {
    return 'KES 0';
  }
  const kes = Math.floor(amountCents / 100);
  return `KES ${kes.toLocaleString('en-KE')}`;
}

export function renderTemplate(
  eventType: string,
  data: NotificationEventData,
  lang: Language = 'en'
): RenderedMessage {
  const orderId = (data.order_id || 'N/A').slice(0, 8);
  const formattedAmount = formatKES(data.amount_cents);
  const receipt = data.provider_reference || 'N/A';
  const reason = data.reason || (lang === 'sw' ? 'sababu za kiufundi' : 'technical reason');

  switch (eventType.toLowerCase()) {
    case 'order.created':
    case 'ordercreated':
      if (lang === 'sw') {
        return {
          smsText: `Habari! Agizo lako la Artisan Market #${orderId} la ${formattedAmount} limepokelewa. Tafadhali thibitisha malipo kupitia M-Pesa kwenye simu yako.`,
          emailSubject: `Artisan Market: Agizo #${orderId} Limepokelewa`,
          emailText: `Habari,\n\nAgizo lako #${orderId} la thamani ya ${formattedAmount} limepokelewa kikamilifu.\n\nTafadhali thibitisha ombi la malipo kwenye simu yako ili kukamilisha ununuzi.\n\nAsante kwa kuunga mkono kazi za mikono za Kenya!`,
          emailHtml: `
            <div style="font-family: Arial, sans-serif; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
              <h2 style="color: #b45309;">Artisan Market</h2>
              <h3>Agizo Lako Limepokelewa</h3>
              <p>Habari,</p>
              <p>Agizo lako <strong>#${orderId}</strong> la thamani ya <strong>${formattedAmount}</strong> limepokelewa.</p>
              <p>Tafadhali thibitisha ombi la malipo la M-Pesa kwenye simu yako.</p>
              <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
              <p style="font-size: 12px; color: #777;">Asante kwa kuwasaidia mafundi wa nchini Kenya.</p>
            </div>
          `,
        };
      }
      return {
        smsText: `Habari! Your Artisan Market order #${orderId} for ${formattedAmount} has been received. Please authorize the M-Pesa prompt on your phone.`,
        emailSubject: `Artisan Market: Order #${orderId} Received`,
        emailText: `Hello,\n\nYour order #${orderId} for ${formattedAmount} has been received.\n\nPlease complete payment on your phone to finalize your order.\n\nThank you for supporting Kenyan artisans!`,
        emailHtml: `
          <div style="font-family: Arial, sans-serif; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
            <h2 style="color: #b45309;">Artisan Market</h2>
            <h3>Your Order Has Been Received</h3>
            <p>Hello,</p>
            <p>Your order <strong>#${orderId}</strong> for <strong>${formattedAmount}</strong> has been received.</p>
            <p>Please authorize the payment prompt on your phone to complete your checkout.</p>
            <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
            <p style="font-size: 12px; color: #777;">Thank you for empowering handcrafted artisan communities in Kenya.</p>
          </div>
        `,
      };

    case 'order.confirmed':
    case 'orderconfirmed':
      if (lang === 'sw') {
        return {
          smsText: `Artisan Market: Agizo lako #${orderId} limethibitishwa! Malipo yamefaulu. Mafundi wetu wanaandaa bidhaa zako sasa.`,
          emailSubject: `Artisan Market: Agizo #${orderId} Limethibitishwa!`,
          emailText: `Habari,\n\nHabari njema! Agizo lako #${orderId} limethibitishwa kikamilifu na malipo yamepokelewa.\n\nMafundi wetu wameanza maandalizi ya bidhaa zako za kipekee.\n\nAsante sana!`,
          emailHtml: `
            <div style="font-family: Arial, sans-serif; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
              <h2 style="color: #047857;">Artisan Market</h2>
              <h3>Agizo Limethibitishwa!</h3>
              <p>Habari,</p>
              <p>Agizo lako <strong>#${orderId}</strong> limethibitishwa na malipo yamekamilika.</p>
              <p>Mafundi wetu wameanza kuandaa bidhaa zako kwa upendo na ustadi wa hali ya juu.</p>
              <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
              <p style="font-size: 12px; color: #777;">Artisan Market &bull; Nairobi, Kenya</p>
            </div>
          `,
        };
      }
      return {
        smsText: `Artisan Market: Order #${orderId} confirmed! Payment was successful. Our Kenyan artisans are now preparing your handcrafted goods.`,
        emailSubject: `Artisan Market: Order #${orderId} Confirmed!`,
        emailText: `Hello,\n\nGreat news! Your order #${orderId} has been confirmed.\n\nOur local artisans are now preparing your handcrafted pieces.\n\nAsante sana for shopping with us!`,
        emailHtml: `
          <div style="font-family: Arial, sans-serif; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
            <h2 style="color: #047857;">Artisan Market</h2>
            <h3>Order Confirmed!</h3>
            <p>Hello,</p>
            <p>Your order <strong>#${orderId}</strong> has been confirmed and payment was received.</p>
            <p>Our Kenyan artisans are crafting and preparing your items with great care.</p>
            <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
            <p style="font-size: 12px; color: #777;">Artisan Market &bull; Nairobi, Kenya</p>
          </div>
        `,
      };

    case 'payment.succeeded':
    case 'paymentsucceeded':
      if (lang === 'sw') {
        return {
          smsText: `Artisan Market: Tumepokea ${formattedAmount} kwa agizo #${orderId} (Risiti: ${receipt}). Asante sana!`,
          emailSubject: `Risiti ya Malipo: Agizo #${orderId}`,
          emailText: `Habari,\n\nTumepokea malipo yako ya ${formattedAmount} kwa agizo #${orderId}.\nNambari ya Risiti: ${receipt}\n\nAsante sana kwa kufanya biashara nasi.`,
          emailHtml: `
            <div style="font-family: Arial, sans-serif; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
              <h2 style="color: #047857;">Artisan Market</h2>
              <h3>Risiti ya Malipo</h3>
              <p>Habari,</p>
              <p>Tumepokea malipo ya <strong>${formattedAmount}</strong> kwa agizo <strong>#${orderId}</strong>.</p>
              <p><strong>Nambari ya Risiti:</strong> ${receipt}</p>
              <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
              <p style="font-size: 12px; color: #777;">Asante sana kwa kusaidia biashara za mafundi.</p>
            </div>
          `,
        };
      }
      return {
        smsText: `Artisan Market: Received ${formattedAmount} for order #${orderId} (Receipt: ${receipt}). Asante sana!`,
        emailSubject: `Payment Receipt: Order #${orderId}`,
        emailText: `Hello,\n\nWe received your payment of ${formattedAmount} for order #${orderId}.\nReceipt / Reference: ${receipt}\n\nThank you for shopping on Artisan Market!`,
        emailHtml: `
          <div style="font-family: Arial, sans-serif; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
            <h2 style="color: #047857;">Artisan Market</h2>
            <h3>Payment Received</h3>
            <p>Hello,</p>
            <p>We received your payment of <strong>${formattedAmount}</strong> for order <strong>#${orderId}</strong>.</p>
            <p><strong>Receipt Number:</strong> ${receipt}</p>
            <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
            <p style="font-size: 12px; color: #777;">Asante sana for shopping with Artisan Market.</p>
          </div>
        `,
      };

    case 'order.cancelled':
    case 'ordercancelled':
    case 'payment.failed':
    case 'paymentfailed':
    default:
      if (lang === 'sw') {
        return {
          smsText: `Artisan Market: Agizo #${orderId} halikufanikiwa (${reason}). Tafadhali jaribu tena au wasiliana na kitengo cha huduma.`,
          emailSubject: `Artisan Market: Notisi ya Agizo #${orderId}`,
          emailText: `Habari,\n\nAgizo #${orderId} limesitishwa au halijafanikiwa kwa sababu: ${reason}.\n\nTafadhali jaribu tena au wasiliana nasi kwa usaidizi.`,
          emailHtml: `
            <div style="font-family: Arial, sans-serif; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
              <h2 style="color: #dc2626;">Artisan Market</h2>
              <h3>Notisi ya Agizo</h3>
              <p>Habari,</p>
              <p>Agizo <strong>#${orderId}</strong> halijakamilika kwa sababu: <em>${reason}</em>.</p>
              <p>Kama pesa zilitolewa kimakosa, zitarudishwa kulingana na taratibu za benki/M-Pesa.</p>
              <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
              <p style="font-size: 12px; color: #777;">Artisan Market &bull; Nairobi, Kenya</p>
            </div>
          `,
        };
      }
      return {
        smsText: `Artisan Market: Order #${orderId} was not completed (${reason}). Please retry checkout or contact support.`,
        emailSubject: `Artisan Market: Notice for Order #${orderId}`,
        emailText: `Hello,\n\nOrder #${orderId} could not be completed (${reason}).\n\nPlease retry checkout or contact support if you need assistance.`,
        emailHtml: `
          <div style="font-family: Arial, sans-serif; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
            <h2 style="color: #dc2626;">Artisan Market</h2>
            <h3>Order Update</h3>
            <p>Hello,</p>
            <p>Order <strong>#${orderId}</strong> could not be completed for the following reason: <em>${reason}</em>.</p>
            <p>If you were charged, your funds will be released automatically.</p>
            <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
            <p style="font-size: 12px; color: #777;">Artisan Market &bull; Nairobi, Kenya</p>
          </div>
        `,
      };
  }
}

